package service

import (
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
)

func TestStartPProfDisabledDoesNotStart(t *testing.T) {
	srv, err := startPProf(config.PProf{})
	require.NoError(t, err)
	assert.Nil(t, srv)
}

func TestStartPProfServesDebugEndpoints(t *testing.T) {
	srv, err := startPProf(config.PProf{Enabled: true, Host: "127.0.0.1", Port: 0})
	require.NoError(t, err)
	require.NotNil(t, srv)
	t.Cleanup(func() { assert.NoError(t, srv.Stop()) })

	res, err := http.Get("http://" + srv.Addr() + "/debug/pprof/")
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "goroutine")

	res, err = http.Get("http://" + srv.Addr() + "/debug/pprof/goroutine?debug=1")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestStartPProfDefaultsToLocalhost(t *testing.T) {
	srv, err := startPProf(config.PProf{Enabled: true, Port: 0})
	require.NoError(t, err)
	require.NotNil(t, srv)
	t.Cleanup(func() { assert.NoError(t, srv.Stop()) })

	host, _, err := net.SplitHostPort(srv.Addr())
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", host)
}

func TestConfiguredPProfLifecycle(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, stopConfiguredPProf()) })

	err := startConfiguredPProf(config.PProf{Enabled: true, Host: "127.0.0.1", Port: 0})
	require.NoError(t, err)

	pprofMu.Lock()
	require.NotNil(t, activePProf)
	addr := activePProf.Addr()
	pprofMu.Unlock()

	res, err := http.Get("http://" + addr + "/debug/pprof/")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	err = startConfiguredPProf(config.PProf{})
	require.NoError(t, err)

	pprofMu.Lock()
	defer pprofMu.Unlock()
	assert.Nil(t, activePProf)
}
