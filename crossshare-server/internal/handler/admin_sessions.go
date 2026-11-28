package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/db"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/service"
)

func (h *AdminHandler) writeAccountError(c *gin.Context, err error) {
	var accountErr *service.AdminError
	switch {
	case errors.As(err, &accountErr):
		writeAdminError(c, accountErr.Status, accountErr.Code, accountErr.Message)
	case errors.Is(err, db.ErrNotFound):
		writeAdminError(c, http.StatusNotFound, "not_found", "account or session not found")
	case errors.Is(err, db.ErrUsernameTaken):
		writeAdminError(c, http.StatusConflict, "username_taken", "username already exists")
	case errors.Is(err, db.ErrLastAdmin):
		writeAdminError(c, http.StatusConflict, "last_admin", err.Error())
	default:
		h.logger.Error().Err(err).Msg("account operation failed")
		writeAdminError(c, http.StatusInternalServerError, "account_error", "account operation failed")
	}
}

func (h *AdminHandler) ListSessions(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	session := middleware.CurrentAdminSession(c)
	items, err := h.accounts.ListSessions(c.Request.Context(), admin.ID, session.ID)
	if err != nil {
		h.writeAccountError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sessions": items})
}

func (h *AdminHandler) RevokeSession(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	id := c.Param("id")
	if err := h.accounts.RevokeSession(c.Request.Context(), admin.ID, id); err != nil {
		h.writeAccountError(c, err)
		return
	}
	if id == middleware.CurrentAdminSession(c).ID {
		h.clearAdminCookie(c)
	}
	h.audit.RecordFromGin(c, audit.Event{ActorType: "admin", ActorID: admin.ID, ActorName: admin.Username,
		Action: "auth.session_revoke", ResourceType: "session", ResourceID: id})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
