package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
	"crossshare-server/internal/service"
)

var p2pSessionPattern = regexp.MustCompile(`^[a-zA-Z0-9]{16}$`)

const (
	p2pRoleSender   = "sender"
	p2pRoleReceiver = "receiver"
)

type P2PHandler struct {
	svc    *service.P2PService
	logger zerolog.Logger
}

func NewP2PHandler(svc *service.P2PService, logger zerolog.Logger) *P2PHandler {
	return &P2PHandler{
		svc:    svc,
		logger: logger.With().Str("handler", "p2p").Logger(),
	}
}

type p2pMessageBody struct {
	From    string          `json:"from" binding:"required"`
	To      string          `json:"to" binding:"required"`
	Type    string          `json:"type" binding:"required"`
	Payload json.RawMessage `json:"payload" binding:"required"`
}

func (h *P2PHandler) CreateSession(c *gin.Context) {
	result, err := h.svc.CreateSession(c.Request.Context())
	if err != nil {
		h.respondP2PError(c, err)
		return
	}

	respondSuccess(c, "p2p session created", result)
}

func (h *P2PHandler) PostMessage(c *gin.Context) {
	sessionID := c.Param("session_id")
	if !p2pSessionPattern.MatchString(sessionID) {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	var body p2pMessageBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	body.From = strings.TrimSpace(body.From)
	body.To = strings.TrimSpace(body.To)
	body.Type = strings.TrimSpace(body.Type)
	if !isP2PRole(body.From) || !isP2PRole(body.To) || body.From == body.To || body.Type == "" {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	result, err := h.svc.PostMessage(c.Request.Context(), sessionID, service.P2PMessageRequest{
		From:    body.From,
		To:      body.To,
		Type:    body.Type,
		Payload: body.Payload,
	})
	if err != nil {
		h.respondP2PError(c, err)
		return
	}

	respondSuccess(c, "p2p message accepted", result)
}

func (h *P2PHandler) GetMessages(c *gin.Context) {
	sessionID := c.Param("session_id")
	if !p2pSessionPattern.MatchString(sessionID) {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	to := strings.TrimSpace(c.Query("to"))
	if !isP2PRole(to) {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	after := int64(0)
	if afterStr := c.Query("after"); afterStr != "" {
		value, err := strconv.ParseInt(afterStr, 10, 64)
		if err != nil || value < 0 {
			respondError(c, apperr.ErrInvalidPayload)
			return
		}
		after = value
	}

	wait := 25 * time.Second
	if waitStr := c.Query("wait"); waitStr != "" {
		value, err := strconv.Atoi(waitStr)
		if err != nil || value < 0 {
			respondError(c, apperr.ErrInvalidPayload)
			return
		}
		wait = time.Duration(value) * time.Second
	}

	messages, err := h.svc.WaitMessages(c.Request.Context(), sessionID, to, after, wait)
	if err != nil {
		h.respondP2PError(c, err)
		return
	}

	respondSuccess(c, "p2p messages", model.P2PMessagesResult{Messages: messages})
}

func (h *P2PHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if !p2pSessionPattern.MatchString(sessionID) {
		respondError(c, apperr.ErrInvalidPayload)
		return
	}

	closed, err := h.svc.CloseSession(c.Request.Context(), sessionID)
	if err != nil {
		h.respondP2PError(c, err)
		return
	}
	if !closed {
		respondError(c, apperr.ErrNotFound)
		return
	}

	respondSuccess(c, "p2p session closed", gin.H{"session_id": sessionID, "closed": true})
}

func (h *P2PHandler) respondP2PError(c *gin.Context, err error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		c.Status(http.StatusNoContent)
		return
	}
	if appErr, ok := err.(*apperr.AppError); ok {
		respondError(c, appErr)
		return
	}
	h.logger.Error().Err(err).Msg("p2p request failed")
	respondError(c, apperr.ErrStorageInternal)
}

func isP2PRole(role string) bool {
	return role == p2pRoleSender || role == p2pRoleReceiver
}
