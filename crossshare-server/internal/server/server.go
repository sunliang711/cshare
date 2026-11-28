package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/config"
	"crossshare-server/internal/db"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/settings"
	"crossshare-server/web"
)

type Version string

var Module = fx.Options(
	fx.Provide(middleware.NewRateLimiter),
	fx.Provide(New),
	fx.Invoke(func(*http.Server) {}),
)

type Params struct {
	fx.In

	Lifecycle     fx.Lifecycle
	Version       Version
	Config        *config.Config
	Logger        zerolog.Logger
	Store         *db.Store
	Settings      *settings.Manager
	HealthHandler *handler.HealthHandler
	PushHandler   *handler.PushHandler
	PullHandler   *handler.PullHandler
	P2PHandler    *handler.P2PHandler
	AdminHandler  *handler.AdminHandler
	RateLimiter   *middleware.RateLimiter
}

func NewRouter(p Params) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(p.Logger))
	r.Use(middleware.CORS(p.Config))
	r.Use(p.RateLimiter.Middleware())

	if p.AdminHandler != nil {
		p.AdminHandler.SetVersion(string(p.Version))
		if p.Config != nil {
			p.AdminHandler.SetResetAdmin(p.Config.Server.ResetAdmin)
		}
	}

	authMw := middleware.Auth(p.Settings)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", p.HealthHandler.Health)

		push := v1.Group("/push")
		push.Use(authMw)
		{
			push.POST("/text", p.PushHandler.PushText)
			push.POST("/binary", p.PushHandler.PushBinary)
			push.POST("/files", p.PushHandler.PushFiles)
			push.POST("", p.PushHandler.PushUnified)
		}

		pull := v1.Group("/pull")
		pull.Use(authMw)
		{
			pull.GET("/:key", p.PullHandler.Pull)
			pull.DELETE("/:key", p.PullHandler.Delete)
		}

		p2p := v1.Group("/p2p")
		p2p.Use(authMw)
		{
			p2p.POST("/sessions", p.P2PHandler.CreateSession)
			p2p.GET("/sessions/:session_id/messages", p.P2PHandler.GetMessages)
			p2p.POST("/sessions/:session_id/messages", p.P2PHandler.PostMessage)
			p2p.DELETE("/sessions/:session_id", p.P2PHandler.CloseSession)
		}
	}

	if p.AdminHandler != nil && p.Store != nil {
		adminAuthMw := middleware.AdminAuth(p.Store, p.AdminHandler.JWTSecret())
		adminAPI := r.Group("/admin/api/v1")
		{
			adminAPI.GET("/setup/status", p.AdminHandler.SetupStatus)
			adminAPI.POST("/setup/init", p.AdminHandler.SetupInit)
			adminAPI.POST("/auth/login", p.AdminHandler.Login)

			protected := adminAPI.Group("")
			protected.Use(adminAuthMw, middleware.AdminPermissions())
			{
				protected.POST("/auth/logout", p.AdminHandler.Logout)
				protected.GET("/auth/me", p.AdminHandler.Me)
				protected.GET("/auth/sessions", p.AdminHandler.ListSessions)
				protected.DELETE("/auth/sessions/:id", p.AdminHandler.RevokeSession)
				protected.PUT("/auth/password", p.AdminHandler.ChangePassword)

				active := protected.Group("")
				active.Use(middleware.RequirePasswordChanged())
				{
					active.GET("/dashboard/summary", p.AdminHandler.DashboardSummary)

					active.GET("/settings", p.AdminHandler.GetSettings)
					active.PUT("/settings", p.AdminHandler.UpdateSettings)
					active.GET("/settings/jwt-secret", p.AdminHandler.RevealClientJWTSecret)
					active.POST("/settings/generate-jwt-secret", p.AdminHandler.GenerateClientJWTSecret)

					active.GET("/admins", p.AdminHandler.ListAdmins)
					active.POST("/admins", p.AdminHandler.CreateAdmin)
					active.PUT("/admins/:id", p.AdminHandler.UpdateAdmin)

					active.GET("/audit-logs", p.AdminHandler.ListAuditLogs)
					active.GET("/audit-logs/export", p.AdminHandler.ExportAuditLogs)

					active.GET("/cron-jobs", p.AdminHandler.ListCronJobs)
					active.POST("/cron-jobs/preview", p.AdminHandler.PreviewCronSchedule)
					active.GET("/cron-jobs/:id/secrets", p.AdminHandler.RevealCronJobSecrets)
					active.PUT("/cron-jobs/:id", p.AdminHandler.UpdateCronJob)
					active.POST("/cron-jobs/:id/run", p.AdminHandler.RunCronJob)
					active.GET("/cron-jobs/:id/runs", p.AdminHandler.ListCronJobRuns)
					active.POST("/cron-jobs/:id/test-safekeep", p.AdminHandler.TestSafekeepConnection)

					active.POST("/migration/export", p.AdminHandler.MigrationExport)
					active.POST("/migration/inspect", p.AdminHandler.MigrationInspect)
					active.POST("/migration/import", p.AdminHandler.MigrationImport)
				}
			}
		}
	}

	registerStaticRoutes(r)
	return r
}

func registerStaticRoutes(r *gin.Engine) {
	var staticFS fs.FS
	if sub, err := fs.Sub(web.StaticFiles, "dist"); err == nil {
		if _, statErr := fs.Stat(sub, "index.html"); statErr == nil {
			staticFS = sub
		}
	}
	if staticFS == nil {
		staticFS, _ = fs.Sub(web.StaticFiles, ".")
	}

	fileServer := http.FileServer(http.FS(staticFS))

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/admin/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "endpoint not found"}})
			return
		}

		reqPath := c.Request.URL.Path
		clean := strings.TrimPrefix(path.Clean(reqPath), "/")
		if clean == "." {
			clean = ""
		}

		if reqPath == "/admin" || strings.HasPrefix(reqPath, "/admin/") {
			if clean != "" {
				if info, err := fs.Stat(staticFS, clean); err == nil && !info.IsDir() {
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
			if data, err := fs.ReadFile(staticFS, "admin/index.html"); err == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
				return
			}
			c.String(http.StatusNotFound, "admin UI not built")
			return
		}

		if clean != "" {
			if info, err := fs.Stat(staticFS, clean); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		if data, err := fs.ReadFile(staticFS, "index.html"); err == nil {
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

func New(p Params) *http.Server {
	r := NewRouter(p)
	addr := fmt.Sprintf(":%d", p.Config.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if p.AdminHandler != nil {
				if _, err := p.AdminHandler.BootstrapReset(ctx, os.Stdout); err != nil {
					return err
				}
			}
			p.Logger.Info().Str("version", string(p.Version)).Str("addr", addr).Msg("starting HTTP server")
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
