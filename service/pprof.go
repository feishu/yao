package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

const (
	defaultPProfHost      = "127.0.0.1"
	pprofShutdownTimeout  = 2 * time.Second
	pprofDiagnosticsRoute = "/debug/pprof/"
)

var (
	pprofMu     sync.Mutex
	activePProf *pprofServer
)

type pprofServer struct {
	server   *http.Server
	listener net.Listener
	done     chan error
	stopOnce sync.Once
	stopErr  error
}

func startConfiguredPProf(cfg config.PProf) error {
	pprofMu.Lock()
	defer pprofMu.Unlock()

	if activePProf != nil {
		if err := activePProf.Stop(); err != nil {
			return err
		}
		activePProf = nil
	}

	srv, err := startPProf(cfg)
	if err != nil {
		return err
	}
	activePProf = srv
	return nil
}

func stopConfiguredPProf() error {
	pprofMu.Lock()
	defer pprofMu.Unlock()

	if activePProf == nil {
		return nil
	}

	err := activePProf.Stop()
	activePProf = nil
	return err
}

func startPProf(cfg config.PProf) (*pprofServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = defaultPProfHost
	}

	addr := net.JoinHostPort(host, strconv.Itoa(cfg.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: newPProfMux(),
	}
	diagnostics := &pprofServer{
		server:   server,
		listener: listener,
		done:     make(chan error, 1),
	}

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			diagnostics.done <- err
			return
		}
		diagnostics.done <- nil
	}()

	log.Info("[PProf] http://%s%s", listener.Addr().String(), pprofDiagnosticsRoute)
	return diagnostics, nil
}

func newPProfMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(pprofDiagnosticsRoute, pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
	}
	return mux
}

func (srv *pprofServer) Addr() string {
	if srv == nil || srv.listener == nil {
		return ""
	}
	return srv.listener.Addr().String()
}

func (srv *pprofServer) Stop() error {
	if srv == nil {
		return nil
	}

	srv.stopOnce.Do(func() {
		srv.stopErr = srv.shutdown()
	})
	return srv.stopErr
}

func (srv *pprofServer) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), pprofShutdownTimeout)
	defer cancel()

	err := srv.server.Shutdown(ctx)
	select {
	case serveErr := <-srv.done:
		if err == nil {
			err = serveErr
		}
	case <-ctx.Done():
		if err == nil {
			err = ctx.Err()
		}
	}
	return err
}
