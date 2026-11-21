package handler

import (
	"errors"
	"net/http"
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
	store     *db.Store
	cipher    *crypto.Cipher
	settings  *settings.Manager
	audit     *audit.Recorder
	sched     *scheduler.Engine
	mig       *migration.Service
	storage   storage.Storage
	p2p       *service.P2PService
	jwtSecret []byte
	version   string
	logger    zerolog.Logger
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
	logger zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
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
		"initialized": n > 0,
		"version":     h.version,
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

	tok, err := middleware.SignAdminToken(h.jwtSecret, admin)
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
	c.JSON(http.StatusCreated, gin.H{"admin": admin, "token": tok})
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
	tok, err := middleware.SignAdminToken(h.jwtSecret, admin)
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
	c.JSON(http.StatusOK, gin.H{"admin": admin, "token": tok})
}

func (h *AdminHandler) Logout(c *gin.Context) {
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
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.AdminCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPSRequest(c.Request),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AdminHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"admin":   middleware.CurrentAdmin(c),
		"version": h.version,
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
	if !crypto.VerifyPassword(admin.PasswordHash, req.OldPassword) {
		writeAdminError(c, http.StatusBadRequest, "invalid_password", "current password is incorrect")
		return
	}
	if len(req.NewPassword) < 8 {
		writeAdminError(c, http.StatusBadRequest, "weak_password", "new password must be at least 8 characters")
		return
	}
	hash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "hash_error", err.Error())
		return
	}
	if err := h.store.UpdateAdminPassword(c.Request.Context(), admin.ID, hash); err != nil {
		writeAdminError(c, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	admin.PasswordHash = hash
	if tok, err := middleware.SignAdminToken(h.jwtSecret, admin); err == nil {
		h.setAdminCookie(c, tok)
	}
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "admin.change_password",
		ResourceType: "admin",
		ResourceID:   admin.ID,
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsActive *bool  `json:"is_active"`
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
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	a := &db.Admin{
		ID:           crypto.NewULID(),
		Username:     req.Username,
		PasswordHash: hash,
		IsActive:     active,
	}
	if err := h.store.CreateAdmin(c.Request.Context(), a); err != nil {
		writeAdminError(c, http.StatusConflict, "create_failed", err.Error())
		return
	}
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      actor.ID,
		ActorName:    actor.Username,
		Action:       "admin.create",
		ResourceType: "admin",
		ResourceID:   a.ID,
		Detail:       gin.H{"username": a.Username, "is_active": a.IsActive},
	})
	c.JSON(http.StatusCreated, a)
}

func (h *AdminHandler) UpdateAdmin(c *gin.Context) {
	actor := middleware.CurrentAdmin(c)
	id := c.Param("id")
	var req struct {
		IsActive bool   `json:"is_active"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if id == actor.ID && !req.IsActive {
		writeAdminError(c, http.StatusBadRequest, "cannot_disable_self", "cannot disable the currently logged-in admin")
		return
	}
	hash := ""
	if req.Password != "" {
		if len(req.Password) < 8 {
			writeAdminError(c, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")
			return
		}
		var err error
		hash, err = crypto.HashPassword(req.Password)
		if err != nil {
			writeAdminError(c, http.StatusInternalServerError, "hash_error", err.Error())
			return
		}
	}
	if err := h.store.UpdateAdmin(c.Request.Context(), id, req.IsActive, hash); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeAdminError(c, http.StatusNotFound, "not_found", "admin not found")
			return
		}
		writeAdminError(c, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      actor.ID,
		ActorName:    actor.Username,
		Action:       "admin.update",
		ResourceType: "admin",
		ResourceID:   id,
		Detail:       gin.H{"is_active": req.IsActive, "password_reset": hash != ""},
	})
	updated, _ := h.store.GetAdminByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, updated)
}
