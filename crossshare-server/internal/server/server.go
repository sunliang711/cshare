package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/config"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/middleware"
)

var Module = fx.Options(
	fx.Provide(middleware.NewRateLimiter),
	fx.Provide(New),
	fx.Invoke(func(*http.Server) {}),
)

type Params struct {
	fx.In

	Lifecycle     fx.Lifecycle
	Config        *config.Config
	Logger        zerolog.Logger
	HealthHandler *handler.HealthHandler
	PushHandler   *handler.PushHandler
	PullHandler   *handler.PullHandler
	RateLimiter   *middleware.RateLimiter
}

func New(p Params) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(p.Logger))
	r.Use(middleware.CORS(p.Config))
	r.Use(p.RateLimiter.Middleware())

	authMw := middleware.Auth(p.Config)

	v2 := r.Group("/api/v1")
	{
		v2.GET("/health", p.HealthHandler.Health)

		push := v2.Group("/push")
		push.Use(authMw)
		{
			push.POST("/text", p.PushHandler.PushText)
			push.POST("/binary", p.PushHandler.PushBinary)
			push.POST("", p.PushHandler.PushUnified)
		}

		pull := v2.Group("/pull")
		pull.Use(authMw)
		{
			pull.GET("/:key", p.PullHandler.Pull)
			pull.DELETE("/:key", p.PullHandler.Delete)
		}
	}

	addr := fmt.Sprintf(":%d", p.Config.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.Logger.Info().Str("addr", addr).Msg("starting HTTP server")
			go func() {
				var err error
				if p.Config.Server.TLSEnable {
					err = srv.ListenAndServeTLS(p.Config.Server.CrtPath, p.Config.Server.KeyPath)
				} else {
					err = srv.ListenAndServe()
				}
				if err != nil && err != http.ErrServerClosed {
					p.Logger.Fatal().Err(err).Msg("server listen error")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info().Msg("shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
