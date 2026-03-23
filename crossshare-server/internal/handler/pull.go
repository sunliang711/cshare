package handler

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
	"crossshare-server/internal/service"
)

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9]{6,12}$`)

type PullHandler struct {
	svc    *service.ShareService
	logger zerolog.Logger
}

func NewPullHandler(svc *service.ShareService, logger zerolog.Logger) *PullHandler {
	return &PullHandler{
		svc:    svc,
		logger: logger.With().Str("handler", "pull").Logger(),
	}
}

func (h *PullHandler) Pull(c *gin.Context) {
	key := c.Param("key")
	if !keyPattern.MatchString(key) {
		respondError(c, apperr.ErrInvalidKey)
		return
	}

	share, err := h.svc.Pull(c.Request.Context(), key)
	if err != nil {
		if appErr, ok := err.(*apperr.AppError); ok {
			respondError(c, appErr)
			return
		}
		respondError(c, apperr.ErrStorageInternal)
		return
	}

	deleteAfterPull := strings.EqualFold(c.GetHeader("Delete-After-Pull"), "true")
	deleted := false
	if deleteAfterPull {
		deleted, _ = h.svc.Delete(c.Request.Context(), key)
	}

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") && share.Type == "text" {
		respondSuccess(c, "pull success", model.PullTextResult{
			Key:         share.Key,
			Text:        string(share.Content),
			Filename:    share.Name,
			ContentType: share.ContentType,
			Size:        share.ContentSize,
			Deleted:     deleted,
		})
		return
	}

	shareType := "Text"
	if share.Type == "binary" {
		shareType = "File"
	}

	c.Header("Crossshare-Type", shareType)
	if share.Name != "" {
		c.Header("Crossshare-Filename", share.Name)
	}
	if deleted {
		c.Header("Key-Deleted", "true")
	} else {
		c.Header("Key-Deleted", "false")
	}
	c.Header("Access-Control-Expose-Headers", "Crossshare-Type,Crossshare-Filename,Key-Deleted,Content-Type")
	c.Data(200, share.ContentType, share.Content)
}

func (h *PullHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if !keyPattern.MatchString(key) {
		respondError(c, apperr.ErrInvalidKey)
		return
	}

	deleted, err := h.svc.Delete(c.Request.Context(), key)
	if err != nil {
		if appErr, ok := err.(*apperr.AppError); ok {
			respondError(c, appErr)
			return
		}
		respondError(c, apperr.ErrStorageInternal)
		return
	}

	if !deleted {
		respondError(c, apperr.ErrNotFound)
		return
	}

	respondSuccess(c, "delete success", model.DeleteResult{
		Key:     key,
		Deleted: true,
	})
}
