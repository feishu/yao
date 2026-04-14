package browser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var browserPathEnvKeys = []string{
	"YAO_BROWSER_BIN",
	"GOOGLE_CHROME_BIN",
	"CHROME_BIN",
	"CHROMIUM_BIN",
}

var browserPathCandidates = []string{
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	"/usr/bin/google-chrome",
	"/usr/bin/chromium",
	"/usr/bin/chromium-browser",
	"/opt/homebrew/bin/chromium",
	"/snap/bin/chromium",
}

func init() {
	pdfRenderer = renderPDFWithRod
	pngRenderer = renderPNGWithRod
}

func renderPDFWithRod(html string, options Options) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(options.Timeout)*time.Millisecond)
	defer cancel()

	page, cleanup, err := newRodPage(ctx, options)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := preparePage(page, html, options); err != nil {
		return nil, err
	}

	req := &proto.PagePrintToPDF{
		Landscape:       options.Landscape,
		PrintBackground: options.PrintBackground,
	}

	if options.Scale > 0 {
		scale := options.Scale
		req.Scale = &scale
	}

	reader, err := page.PDF(req)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func renderPNGWithRod(html string, options Options) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(options.Timeout)*time.Millisecond)
	defer cancel()

	page, cleanup, err := newRodPage(ctx, options)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := preparePage(page, html, options); err != nil {
		return nil, err
	}

	req := &proto.PageCaptureScreenshot{
		Format:                proto.PageCaptureScreenshotFormatPng,
		FromSurface:           true,
		CaptureBeyondViewport: options.FullPage,
	}

	return page.Screenshot(options.FullPage, req)
}

func newRodPage(ctx context.Context, options Options) (*rod.Page, func(), error) {
	launch := launcher.New().Context(ctx).Headless(true)

	bin, err := discoverBrowserBinary()
	if err != nil {
		return nil, nil, err
	}
	if bin != "" {
		launch.Bin(bin)
	}

	controlURL, err := launch.Launch()
	if err != nil {
		return nil, nil, err
	}

	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		launch.Cleanup()
		return nil, nil, err
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		_ = browser.Close()
		launch.Cleanup()
		return nil, nil, err
	}

	cleanup := func() {
		_ = page.Close()
		_ = browser.Close()
		launch.Cleanup()
	}

	return page, cleanup, nil
}

func preparePage(page *rod.Page, html string, options Options) error {
	scale := 1.0
	if options.Scale > 0 {
		scale = options.Scale
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             options.Width,
		Height:            options.Height,
		DeviceScaleFactor: scale,
		Mobile:            false,
	}); err != nil {
		return err
	}

	if err := applyDefaultRenderingEnvironment(page); err != nil {
		return err
	}

	if err := page.SetDocumentContent(applyBaseURL(html, options.BaseURL)); err != nil {
		return err
	}

	if err := page.WaitLoad(); err != nil {
		return err
	}

	if options.Wait > 0 {
		select {
		case <-page.GetContext().Done():
			return page.GetContext().Err()
		case <-time.After(time.Duration(options.Wait) * time.Millisecond):
		}
	}

	return nil
}

func applyDefaultRenderingEnvironment(page *rod.Page) error {
	media := proto.EmulationSetEmulatedMedia{
		Features: []*proto.EmulationMediaFeature{
			{Name: "prefers-color-scheme", Value: "light"},
		},
	}
	if err := media.Call(page); err != nil {
		return err
	}

	alpha := 1.0
	background := proto.EmulationSetDefaultBackgroundColorOverride{
		Color: &proto.DOMRGBA{
			R: 255,
			G: 255,
			B: 255,
			A: &alpha,
		},
	}
	return background.Call(page)
}

func discoverBrowserBinary() (string, error) {
	for _, key := range browserPathEnvKeys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}

		if _, err := os.Stat(value); err != nil {
			return "", fmt.Errorf("browser binary configured by %s is not accessible: %s", key, value)
		}

		return value, nil
	}

	if path, has := launcher.LookPath(); has && path != "" {
		return path, nil
	}

	for _, path := range browserPathCandidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", nil
}

func applyBaseURL(html string, baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return html
	}

	baseTag := fmt.Sprintf(`<base href="%s">`, baseURL)
	lower := strings.ToLower(html)
	headIndex := strings.Index(lower, "<head")
	if headIndex >= 0 {
		tagEnd := strings.Index(html[headIndex:], ">")
		if tagEnd >= 0 {
			insertAt := headIndex + tagEnd + 1
			return html[:insertAt] + baseTag + html[insertAt:]
		}
	}

	htmlIndex := strings.Index(lower, "<html")
	if htmlIndex >= 0 {
		tagEnd := strings.Index(html[htmlIndex:], ">")
		if tagEnd >= 0 {
			insertAt := htmlIndex + tagEnd + 1
			return html[:insertAt] + "<head>" + baseTag + "</head>" + html[insertAt:]
		}
	}

	bodyIndex := strings.Index(lower, "<body")
	if bodyIndex >= 0 {
		tagEnd := strings.Index(html[bodyIndex:], ">")
		if tagEnd >= 0 {
			insertAt := bodyIndex + tagEnd + 1
			return html[:insertAt] + baseTag + html[insertAt:]
		}
	}

	return "<html><head>" + baseTag + "</head><body>" + html + "</body></html>"
}

func isAbsolutePath(name string) bool {
	return filepath.IsAbs(name)
}
