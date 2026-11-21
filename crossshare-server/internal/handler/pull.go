package handler

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"crossshare-server/internal/audit"
	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
	"crossshare-server/internal/service"
)

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9]{6,12}$`)

type PullHandler struct {
	svc    *service.ShareService
	audit  *audit.Recorder
	logger zerolog.Logger
}

func NewPullHandler(svc *service.ShareService, auditRec *audit.Recorder, logger zerolog.Logger) *PullHandler {
	return &PullHandler{
		svc:    svc,
		audit:  auditRec,
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
		h.audit.RecordFromGin(c, audit.Event{
			ActorType:    "client",
			ActorName:    c.GetString("creator"),
			Action:       "pull",
			ResourceType: "share",
			ResourceID:   key,
			ShareKey:     key,
			Status:       "failure",
			Detail:       map[string]any{"error": err.Error()},
		})
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

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "client",
		ActorName:    c.GetString("creator"),
		Action:       "pull",
		ResourceType: "share",
		ResourceID:   key,
		ShareKey:     key,
		Detail: map[string]any{
			"type":              share.Type,
			"size":              share.ContentSize,
			"delete_after_pull": deleteAfterPull,
			"deleted":           deleted,
		},
	})

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		if share.Type == "text" {
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
		if share.Type == "files" {
			respondSuccess(c, "pull success", model.PullFilesResult{
				Key:         share.Key,
				Filename:    share.Name,
				ContentType: share.ContentType,
				Size:        share.ContentSize,
				StoredSize:  share.StoredSize,
				FileCount:   len(share.Files),
				Files:       share.Files,
				Deleted:     deleted,
			})
			return
		}
	}

	shareType := "Text"
	contentType := share.ContentType
	content := share.Content
	filename := share.Name
	fileCount := len(share.Files)

	if share.Type == "files" {
		if len(share.Files) == 1 {
			file, err := h.svc.PullSingleFile(share)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					respondError(c, appErr)
					return
				}
				respondError(c, apperr.ErrStorageInternal)
				return
			}
			shareType = "File"
			contentType = file.ContentType
			content = file.Data
			filename = file.Filename
		} else {
			shareType = "Bundle"
			if filename == "" {
				filename = "crossshare-files.zip"
			}
			contentType = "application/zip"
		}
	}

	c.Header("Crossshare-Type", shareType)
	if filename != "" {
		c.Header("Crossshare-Filename", filename)
	}
	if share.Type == "files" {
		c.Header("Crossshare-File-Count", strconv.Itoa(fileCount))
	}
	if deleted {
		c.Header("Key-Deleted", "true")
	} else {
		c.Header("Key-Deleted", "false")
	}
	c.Header("Access-Control-Expose-Headers", "Crossshare-Type,Crossshare-Filename,Crossshare-File-Count,Key-Deleted,Content-Type")
	c.Data(200, contentType, content)
}

func (h *PullHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if !keyPattern.MatchString(key) {
		respondError(c, apperr.ErrInvalidKey)
		return
	}

	deleted, err := h.svc.Delete(c.Request.Context(), key)
	if err != nil {
		h.audit.RecordFromGin(c, audit.Event{
			ActorType:    "client",
			ActorName:    c.GetString("creator"),
			Action:       "delete",
			ResourceType: "share",
			ResourceID:   key,
			ShareKey:     key,
			Status:       "failure",
			Detail:       map[string]any{"error": err.Error()},
		})
		if appErr, ok := err.(*apperr.AppError); ok {
			respondError(c, appErr)
			return
		}
		respondError(c, apperr.ErrStorageInternal)
		return
	}

	if !deleted {
		h.audit.RecordFromGin(c, audit.Event{
			ActorType:    "client",
			ActorName:    c.GetString("creator"),
			Action:       "delete",
			ResourceType: "share",
			ResourceID:   key,
			ShareKey:     key,
			Status:       "failure",
			Detail:       map[string]any{"error": "share not found"},
		})
		respondError(c, apperr.ErrNotFound)
		return
	}

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "client",
		ActorName:    c.GetString("creator"),
		Action:       "delete",
		ResourceType: "share",
		ResourceID:   key,
		ShareKey:     key,
	})
	respondSuccess(c, "delete success", model.DeleteResult{
		Key:     key,
		Deleted: true,
	})
}
