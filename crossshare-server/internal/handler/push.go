package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"crossshare-server/internal/audit"
	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/service"
	"crossshare-server/internal/settings"
)

type PushHandler struct {
	svc      *service.ShareService
	settings *settings.Manager
	audit    *audit.Recorder
	logger   zerolog.Logger
}

func NewPushHandler(svc *service.ShareService, settingsMgr *settings.Manager, auditRec *audit.Recorder, logger zerolog.Logger) *PushHandler {
	return &PushHandler{
		svc:      svc,
		settings: settingsMgr,
		audit:    auditRec,
		logger:   logger.With().Str("handler", "push").Logger(),
	}
}

type pushTextBody struct {
	Text        string `json:"text" binding:"required"`
	TTL         int    `json:"ttl"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

const multipartOverheadLimit = 1 << 20

func (h *PushHandler) PushText(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.settings.Get().TextJSONLimit+4096)

	var body pushTextBody
	if err := c.ShouldBindJSON(&body); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(c, apperr.ErrPayloadTooLarge)
			return
		}
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	if strings.TrimSpace(body.Text) == "" {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	result, err := h.svc.PushText(c.Request.Context(), &service.PushTextRequest{
		Text:        body.Text,
		TTL:         body.TTL,
		Filename:    body.Filename,
		ContentType: body.ContentType,
		Creator:     c.GetString("creator"),
	})
	if err != nil {
		h.auditPushFailure(c, "push.text", err)
		if appErr, ok := err.(*apperr.AppError); ok {
			respondError(c, appErr)
			return
		}
		respondError(c, apperr.ErrStorageInternal)
		return
	}

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "client",
		ActorName:    c.GetString("creator"),
		Action:       "push.text",
		ResourceType: "share",
		ResourceID:   result.Key,
		ShareKey:     result.Key,
		Detail: map[string]any{
			"type": result.Type,
			"size": result.Size,
			"ttl":  result.TTL,
		},
	})
	respondSuccess(c, "push success", result)
}

func (h *PushHandler) PushBinary(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.settings.Get().FilesPushLimit+1)

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(c, apperr.ErrPayloadTooLarge)
			return
		}
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	if len(data) == 0 {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	contentType := c.GetHeader("X-Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	ttl := 0
	if ttlStr := c.GetHeader("X-TTL"); ttlStr != "" {
		ttl, _ = strconv.Atoi(ttlStr)
	}

	result, err := h.svc.PushBinary(c.Request.Context(), &service.PushBinaryRequest{
		Data:        data,
		TTL:         ttl,
		Filename:    c.GetHeader("Filename"),
		ContentType: contentType,
		Creator:     c.GetString("creator"),
	})
	if err != nil {
		h.auditPushFailure(c, "push.binary", err)
		if appErr, ok := err.(*apperr.AppError); ok {
			respondError(c, appErr)
			return
		}
		respondError(c, apperr.ErrStorageInternal)
		return
	}

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "client",
		ActorName:    c.GetString("creator"),
		Action:       "push.binary",
		ResourceType: "share",
		ResourceID:   result.Key,
		ShareKey:     result.Key,
		Detail: map[string]any{
			"type":     result.Type,
			"size":     result.Size,
			"ttl":      result.TTL,
			"filename": result.Filename,
		},
	})
	respondSuccess(c, "push success", result)
}

func (h *PushHandler) PushFiles(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.settings.Get().FilesPushLimit+multipartOverheadLimit)

	form, err := c.MultipartForm()
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(c, apperr.ErrPayloadTooLarge)
			return
		}
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	headers := form.File["files"]
	if len(headers) == 0 {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	files := make([]service.PushFileInput, 0, len(headers))
	for _, header := range headers {
		f, err := header.Open()
		if err != nil {
			respondError(c, apperr.ErrInvalidPayload)
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			respondError(c, apperr.ErrInvalidPayload)
			return
		}

		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		files = append(files, service.PushFileInput{
			Data:        data,
			Filename:    header.Filename,
			ContentType: contentType,
		})
	}

	ttl := 0
	if ttlStr := c.PostForm("ttl"); ttlStr != "" {
		ttl, _ = strconv.Atoi(ttlStr)
	}

	result, err := h.svc.PushFiles(c.Request.Context(), &service.PushFilesRequest{
		Files:   files,
		TTL:     ttl,
		Name:    c.PostForm("name"),
		Creator: c.GetString("creator"),
	})
	if err != nil {
		h.auditPushFailure(c, "push.files", err)
		if appErr, ok := err.(*apperr.AppError); ok {
			respondError(c, appErr)
			return
		}
		respondError(c, apperr.ErrStorageInternal)
		return
	}

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "client",
		ActorName:    c.GetString("creator"),
		Action:       "push.files",
		ResourceType: "share",
		ResourceID:   result.Key,
		ShareKey:     result.Key,
		Detail: map[string]any{
			"type":       result.Type,
			"size":       result.Size,
			"file_count": result.FileCount,
			"ttl":        result.TTL,
			"filename":   result.Filename,
		},
	})
	respondSuccess(c, "push success", result)
}

func (h *PushHandler) auditPushFailure(c *gin.Context, action string, err error) {
	if h.audit == nil {
		return
	}
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "client",
		ActorName:    c.GetString("creator"),
		Action:       action,
		ResourceType: "share",
		Status:       "failure",
		Detail:       map[string]any{"error": err.Error()},
	})
}

func (h *PushHandler) PushUnified(c *gin.Context) {
	ct := c.ContentType()
	switch {
	case strings.HasPrefix(ct, "application/json"):
		h.PushText(c)
	case strings.HasPrefix(ct, "application/octet-stream"):
		h.PushBinary(c)
	case strings.HasPrefix(ct, "multipart/form-data"):
		h.PushFiles(c)
	default:
		respondError(c, apperr.ErrUnsupportedType)
	}
}
