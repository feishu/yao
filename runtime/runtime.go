package runtime

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
)

// Start v8 runtime
func Start(cfg config.Config) error {

	debug := false
	if cfg.Mode == "development" {
		debug = true
	}

	inspect, err := inspectFromConfig(cfg)
	if err != nil {
		return err
	}

	option := &v8.Option{
		MinSize:           cfg.Runtime.MinSize,
		MaxSize:           cfg.Runtime.MaxSize,
		HeapSizeLimit:     cfg.Runtime.HeapSizeLimit,
		HeapAvailableSize: cfg.Runtime.HeapAvailableSize,
		HeapSizeRelease:   cfg.Runtime.HeapSizeRelease,
		Precompile:        cfg.Runtime.Precompile,
		DataRoot:          cfg.DataRoot,
		Mode:              cfg.Runtime.Mode,
		DefaultTimeout:    cfg.Runtime.DefaultTimeout,
		ContextTimeout:    cfg.Runtime.ContextTimeout,
		Import:            cfg.Runtime.Import,
		Debug:             debug,
		ConsoleMode:       cfg.Mode,
		Inspect:           inspect,
	}

	// Read the tsconfig.json
	if cfg.Runtime.Import && application.App != nil {
		if exist, _ := application.App.Exists("tsconfig.json"); exist {
			var tsconfig v8.TSConfig
			raw, err := application.App.Read("tsconfig.json")
			if err != nil {
				return fmt.Errorf("tsconfig.json is not a valid json file %s", err)
			}

			err = jsoniter.Unmarshal(raw, &tsconfig)
			if err != nil {
				return fmt.Errorf("tsconfig.json is not a valid json file %s", err)
			}
			option.TSConfig = &tsconfig
		}
	}

	err = v8.Start(option)
	if err != nil {
		return err
	}

	return nil
}

func inspectFromConfig(cfg config.Config) (v8.Inspect, error) {
	return parseInspect(cfg.Runtime.Inspect, cfg.Mode, cfg.Runtime.InspectTrace, cfg.Runtime.InspectTracePath, cfg.Runtime.InspectSourceContent)
}

func parseInspect(value string, mode string, trace bool, tracePath string, exposeSourceContent *bool) (v8.Inspect, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return v8.Inspect{}, nil
	}
	if mode != "development" {
		return v8.Inspect{}, fmt.Errorf("runtime inspect is only allowed in development mode")
	}

	host := "127.0.0.1"
	portText := value
	if strings.Contains(value, ":") {
		parsedHost, parsedPort, err := net.SplitHostPort(value)
		if err != nil {
			parts := strings.Split(value, ":")
			if len(parts) != 2 {
				return v8.Inspect{}, fmt.Errorf("invalid inspect address %q", value)
			}
			parsedHost, parsedPort = parts[0], parts[1]
		}
		if parsedHost != "" {
			host = parsedHost
		}
		portText = parsedPort
	}

	if !isLocalInspectHost(host) {
		return v8.Inspect{}, fmt.Errorf("inspect host must be localhost, got %s", host)
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return v8.Inspect{}, fmt.Errorf("invalid inspect port %q", portText)
	}

	return v8.Inspect{
		Enabled:             true,
		Host:                host,
		Port:                port,
		Trace:               trace,
		TracePath:           tracePath,
		ExposeSourceContent: exposeSourceContent,
	}, nil
}

func isLocalInspectHost(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// Stop v8 runtime
func Stop() error {
	v8.Stop()
	return nil
}
