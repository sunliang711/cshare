package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"crossshare-server/internal/model"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	respondSuccess(c, "OK", model.HealthResult{
		Service: "crossshare-server",
		Status:  "up",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}
