package runtime

import (
	"os"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
)

func TestParseInspectDevelopmentAddress(t *testing.T) {
	inspect, err := parseInspect("127.0.0.1:9229", "development", false, "", boolPtr(true))
	if err != nil {
		t.Fatal(err)
	}

	if !inspect.Enabled {
		t.Fatal("expected inspect to be enabled")
	}
	if inspect.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %s", inspect.Host)
	}
	if inspect.Port != 9229 {
		t.Fatalf("expected port 9229, got %d", inspect.Port)
	}
}

func TestParseInspectPortOnly(t *testing.T) {
	inspect, err := parseInspect("9230", "development", false, "", boolPtr(true))
	if err != nil {
		t.Fatal(err)
	}

	if inspect.Host != "127.0.0.1" || inspect.Port != 9230 {
		t.Fatalf("unexpected inspect address: %+v", inspect)
	}
}

func TestParseInspectRejectsProduction(t *testing.T) {
	if _, err := parseInspect("127.0.0.1:9229", "production", false, "", boolPtr(true)); err == nil {
		t.Fatal("expected production inspect to fail")
	}
}

func TestParseInspectRejectsNonLocalHost(t *testing.T) {
	if _, err := parseInspect("0.0.0.0:9229", "development", false, "", boolPtr(true)); err == nil {
		t.Fatal("expected non-local inspect host to fail")
	}
}

func TestParseInspectTraceOptions(t *testing.T) {
	inspect, err := parseInspect("127.0.0.1:9229", "development", true, "/tmp/custom-cdp.log", boolPtr(true))
	if err != nil {
		t.Fatal(err)
	}
	if !inspect.Trace || inspect.TracePath != "/tmp/custom-cdp.log" {
		t.Fatalf("unexpected inspect trace options: %+v", inspect)
	}
}

func TestParseInspectSourceContentOption(t *testing.T) {
	inspect, err := parseInspect("127.0.0.1:9229", "development", false, "", boolPtr(false))
	if err != nil {
		t.Fatal(err)
	}
	if inspect.ExposeSourceContent == nil || *inspect.ExposeSourceContent {
		t.Fatal("expected source content exposure to be disabled")
	}
}

func TestParseInspectSourceContentDefaultsOn(t *testing.T) {
	inspect, err := parseInspect("127.0.0.1:9229", "development", false, "", boolPtr(true))
	if err != nil {
		t.Fatal(err)
	}
	if inspect.ExposeSourceContent == nil || !*inspect.ExposeSourceContent {
		t.Fatal("expected source content exposure to default on")
	}
}

func TestParseInspectSourceContentUsesPolicyDefaultWhenUnset(t *testing.T) {
	inspect, err := parseInspect("127.0.0.1:9229", "development", false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if inspect.ExposeSourceContent != nil {
		t.Fatal("expected unset source content option to defer to v8 policy default")
	}
}

func TestInspectFromProgrammaticConfigKeepsSourceContentDefault(t *testing.T) {
	inspect, err := inspectFromConfig(config.Config{
		Mode: "development",
		Runtime: config.Runtime{
			Inspect: "127.0.0.1:9229",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inspect.ExposeSourceContent != nil {
		t.Fatal("expected programmatic config default to defer source content policy")
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func TestStart(t *testing.T) {
	testPrepare(t)
	defer Stop()
	err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}

func testPrepare(t *testing.T, rootEnv ...string) {

	appRootEnv := "YAO_TEST_APPLICATION"
	if len(rootEnv) > 0 {
		appRootEnv = rootEnv[0]
	}

	root := os.Getenv(appRootEnv)
	var app application.Application
	var err error

	app, err = application.OpenFromDisk(root) // Load app from Disk
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
}
