package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"llmtrace/internal/appserver"
	"llmtrace/internal/config"
	"llmtrace/internal/platform/xlog"
)

// Run assembles the application and starts the HTTP server.
func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.App.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	log, err := xlog.New(cfg.Log.Level)
	if err != nil {
		return err
	}

	engine, err := appserver.NewEngine(log, appserver.HTTPConfig{
		AppName:        cfg.App.Name,
		TrustedProxies: cfg.App.TrustedProxies,
		HTTPAccess:     cfg.Log.HTTPAccess,
	}, appserver.NewRuntimeService())
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: engine,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		log.Info("server started", "port", cfg.App.Port, "env", cfg.App.Env)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
