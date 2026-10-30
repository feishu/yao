package service

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	gouHttp "github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/test"
)

func TestStartStop(t *testing.T) {

	gin.SetMode(gin.ReleaseMode)

	cfg := config.Conf
	cfg.Port = 0
	err := engine.Load(cfg, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	srv, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop(srv)

	<-srv.Event()
	if !srv.Ready() {
		t.Fatal("server not ready")
	}

	port, err := srv.Port()
	if err != nil {
		t.Fatal(err)
	}

	if port <= 0 {
		t.Fatal("invalid port")
	}

	// API Server
	req := test.NewRequest(port).Route("/api/__yao/app/setting")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	data, err := res.Map()
	if err != nil {
		t.Fatal(err)
	}
	// assert.Equal(t, "Demo Application", data["name"])
	assert.True(t, len(data["name"].(string)) > 0)

	// Public
	req = test.NewRequest(port).Route("/")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	assert.Equal(t, "Hello World\n", res.Body())

	// XGEN
	req = test.NewRequest(port).Route("/admin/")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	assert.Contains(t, res.Body(), "ROOT /admin/")
}

func TestRestartHotSwap(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	initialRouter := gin.New()
	initialRouter.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, "initial")
	})

	srv := gouHttp.New(initialRouter, gouHttp.Option{
		Host:    "127.0.0.1",
		Port:    0,
		Timeout: 200 * time.Millisecond,
	})

	var err error
	go func() { err = srv.Start() }()
	defer Stop(srv)

	<-srv.Event()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, srv.Ready())

	port, err := srv.Port()
	assert.NoError(t, err)

	// Verify initial router
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ping", port))
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	_ = resp.Body.Close()

	// Hot restart using service.Restart
	cfg := config.Conf
	cfg.Port = port
	err = Restart(srv, cfg)
	assert.NoError(t, err)

	// Port should remain identical and server should remain Ready without dropping TCP
	newPort, err := srv.Port()
	assert.NoError(t, err)
	assert.Equal(t, port, newPort)
	assert.True(t, srv.Ready())
}

