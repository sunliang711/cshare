package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/model"
)

func respondJSON(c *gin.Context, httpStatus int, code int, msg string, data interface{}) {
	c.JSON(httpStatus, model.Response{
		Code:      code,
		Msg:       msg,
		Data:      data,
		RequestID: c.GetString("request_id"),
	})
}

func respondSuccess(c *gin.Context, msg string, data interface{}) {
	respondJSON(c, http.StatusOK, 0, msg, data)
}

func respondError(c *gin.Context, err *apperr.AppError) {
	respondJSON(c, err.HTTPStatus, err.Code, err.Message, nil)
}
