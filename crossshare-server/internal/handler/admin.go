package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/scheduler"
	"crossshare-server/internal/service"
	"crossshare-server/internal/settings"
	"crossshare-server/internal/storage"
)

type AdminHandler struct {
	accounts  *service.AdminService
	store     *db.Store
	cipher    *crypto.Cipher
	settings  *settings.Manager
	audit     *audit.Recorder
	sched     *scheduler.Engine
	mig       *migration.Service
	storage   storage.Storage
	p2p        *service.P2PService
	jwtSecret  []byte
	version    string
	resetAdmin bool
	logger     zerolog.Logger
}

func NewAdminHandler(
	store *db.Store,
	cipher *crypto.Cipher,
	settingsMgr *settings.Manager,
	auditRec *audit.Recorder,
	sched *scheduler.Engine,
	mig *migration.Service,
	st storage.Storage,
	p2p *service.P2PService,
	accounts *service.AdminService,
	logger zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
		accounts:  accounts,
		store:     store,
		cipher:    cipher,
		settings:  settingsMgr,
		audit:     auditRec,
		sched:     sched,
		mig:       mig,
		storage:   st,
		p2p:       p2p,
		jwtSecret: middleware.AdminJWTSecret(cipher),
		version:   "dev",
		logger:    logger.With().Str("handler", "admin").Logger(),
	}
}

func (h *AdminHandler) SetVersion(v string) {
	if strings.TrimSpace(v) != "" {
		h.version = v
	}
}

func (h *AdminHandler) SetResetAdmin(enabled bool) {
	h.resetAdmin = enabled
}

// BootstrapReset resets the primary admin password to a temporary password on startup when RESET_ADMIN is active.
func (h *AdminHandler) BootstrapReset(ctx context.Context, out io.Writer) (string, error) {
	if !h.resetAdmin || h.store == nil {
		return "", nil
	}
	if out == nil {
		out = os.Stdout
	}

	tempPassword, err := crypto.GenerateTemporaryPassword(20)
	if err != nil {
		return "", fmt.Errorf("generate temporary admin password: %w", err)
	}
	hash, err := crypto.HashPassword(tempPassword)
	if err != nil {
		return "", fmt.Errorf("hash temporary admin password: %w", err)
	}

	admin, err := h.store.ResetPrimaryAdminPassword(ctx, hash)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			fmt.Fprintf(out, "\n========================================================================\n")
			fmt.Fprintf(out, "  CrossShare - RESET_ADMIN Active (No Admin Accounts Found)\n")
			fmt.Fprintf(out, "  Action Required : Open /admin to create the initial admin account,\n")
			fmt.Fprintf(out, "                    then remove RESET_ADMIN / server.reset_admin.\n")
			fmt.Fprintf(out, "========================================================================\n\n")
			return "", nil
		}
		return "", fmt.Errorf("reset primary admin password: %w", err)
	}

	fmt.Fprintf(out, "\n========================================================================\n")
	fmt.Fprintf(out, "  CrossShare - Web UI Admin Credentials (Admin Password Reset)\n")
	fmt.Fprintf(out, "  Username           : %s\n", admin.Username)
	fmt.Fprintf(out, "  Temporary Password : %s\n", tempPassword)
	fmt.Fprintf(out, "  Action Required    : Sign in to /admin and set a new password,\n")
	fmt.Fprintf(out, "                       then remove RESET_ADMIN / server.reset_admin.\n")
	fmt.Fprintf(out, "========================================================================\n\n")

	if h.audit != nil {
		h.audit.Record(audit.Event{
			ActorType:    "system",
			ActorID:      "bootstrap",
			ActorName:    "system",
			Action:       "admin.password_reset_bootstrap",
			ResourceType: "admin",
			ResourceID:   admin.ID,
			Detail:       map[string]any{"username": admin.Username},
		})
	}
	h.logger.Warn().Str("username", admin.Username).Msg("admin password reset via RESET_ADMIN; temporary password printed to stdout")
	return tempPassword, nil
}

func (h *AdminHandler) JWTSecret() []byte {
	return h.jwtSecret
}

func writeAdminError(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": msg,
		},
	})
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (h *AdminHandler) setAdminCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.AdminCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPSRequest(c.Request),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
}

// --- Setup & Auth ---

func (h *AdminHandler) SetupStatus(c *gin.Context) {
	n, err := h.store.CountAdmins(c.Request.Context())
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"initialized":            n > 0,
		"version":                h.version,
		"reset_admin_env_active": h.resetAdmin,
	})
}

func (h *AdminHandler) SetupInit(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Password) < 8 {
		writeAdminError(c, http.StatusBadRequest, "validation_error", "username >= 3 chars and password >= 8 chars required")
		return
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "hash_error", err.Error())
		return
	}
	admin := &db.Admin{
		ID:           crypto.NewULID(),
		Username:     req.Username,
		PasswordHash: hash,
		IsActive:     true,
	}
	if err := h.store.CreateInitialAdminIfEmpty(c.Request.Context(), admin); err != nil {
		if errors.Is(err, db.ErrAlreadyInitialized) {
			writeAdminError(c, http.StatusConflict, "already_initialized", "admin is already initialized")
			return
		}
		writeAdminError(c, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	tok, err := h.accounts.CreateSession(c.Request.Context(), admin, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "token_error", err.Error())
		return
	}
	h.setAdminCookie(c, tok)
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "system.setup_init",
		ResourceType: "admin",
		ResourceID:   admin.ID,
	})
	c.JSON(http.StatusCreated, gin.H{
		"admin":                  admin,
		"token":                  tok,
		"reset_admin_env_active": h.resetAdmin,
	})
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	username := strings.TrimSpace(req.Username)
	admin, err := h.store.GetAdminByUsername(c.Request.Context(), username)
	if err != nil || !admin.IsActive || !crypto.VerifyPassword(admin.PasswordHash, req.Password) {
		h.audit.RecordFromGin(c, audit.Event{
			ActorType:    "admin",
			ActorName:    username,
			Action:       "admin.login",
			ResourceType: "admin",
			Status:       "failure",
		})
		writeAdminError(c, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}

	_ = h.store.UpdateAdminLogin(c.Request.Context(), admin.ID)
	tok, err := h.accounts.CreateSession(c.Request.Context(), admin, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "token_error", err.Error())
		return
	}
	h.setAdminCookie(c, tok)
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "admin.login",
		ResourceType: "admin",
		ResourceID:   admin.ID,
	})
	c.JSON(http.StatusOK, gin.H{
		"admin":                  admin,
		"token":                  tok,
		"reset_admin_env_active": h.resetAdmin,
	})
}

func (h *AdminHandler) Logout(c *gin.Context) {
	session := middleware.CurrentAdminSession(c)
	if session != nil {
		if err := h.accounts.RevokeSession(c.Request.Context(), session.AdminID, session.ID); err != nil {
			h.writeAccountError(c, err)
			return
		}
	}
	if admin := middleware.CurrentAdmin(c); admin != nil {
		h.audit.RecordFromGin(c, audit.Event{
			ActorType:    "admin",
			ActorID:      admin.ID,
			ActorName:    admin.Username,
			Action:       "admin.logout",
			ResourceType: "admin",
			ResourceID:   admin.ID,
		})
	}
	h.clearAdminCookie(c)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AdminHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"admin":                  middleware.CurrentAdmin(c),
		"version":                h.version,
		"reset_admin_env_active": h.resetAdmin,
	})
}

func (h *AdminHandler) ChangePassword(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	tok, err := h.accounts.ChangePassword(c.Request.Context(), admin, req.OldPassword, req.NewPassword, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.writeAccountError(c, err)
		return
	}
	h.setAdminCookie(c, tok)
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "admin.change_password",
		ResourceType: "admin",
		ResourceID:   admin.ID,
	})
	c.JSON(http.StatusOK, gin.H{
		"status":                 "ok",
		"admin":                  admin,
		"reset_admin_env_active": h.resetAdmin,
	})
}

// --- Admins Management ---

func (h *AdminHandler) ListAdmins(c *gin.Context) {
	admins, err := h.store.ListAdmins(c.Request.Context())
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": admins})
}

func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	actor := middleware.CurrentAdmin(c)
	var req service.CreateAdminInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	account, err := h.accounts.CreateAccount(c.Request.Context(), actor, req)
	if err != nil {
		h.writeAccountError(c, err)
		return
	}
	h.audit.RecordFromGin(c, audit.Event{ActorType: "admin", ActorID: actor.ID, ActorName: actor.Username,
		Action: "admin.create", ResourceType: "admin", ResourceID: account.ID,
		Detail: gin.H{"username": account.Username, "role": account.Role, "is_active": account.IsActive}})
	c.JSON(http.StatusCreated, account)
}

func (h *AdminHandler) UpdateAdmin(c *gin.Context) {
	actor := middleware.CurrentAdmin(c)
	var req service.UpdateAdminInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	account, token, err := h.accounts.UpdateAccount(c.Request.Context(), actor, c.Param("id"), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.writeAccountError(c, err)
		return
	}
	if token != "" {
		h.setAdminCookie(c, token)
	} else if account.ID == actor.ID && account.Role != actor.Role {
		h.clearAdminCookie(c)
	}
	h.audit.RecordFromGin(c, audit.Event{ActorType: "admin", ActorID: actor.ID, ActorName: actor.Username,
		Action: "admin.update", ResourceType: "admin", ResourceID: account.ID,
		Detail: gin.H{"is_active": account.IsActive, "role": account.Role, "password_reset": req.Password != ""}})
	c.JSON(http.StatusOK, account)
}

func (h *AdminHandler) clearAdminCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.AdminCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPSRequest(c.Request),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}
