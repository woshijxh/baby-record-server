package handler

import (
	"strconv"

	"baby-record-server/internal/apperror"
	"baby-record-server/internal/middleware"
	"github.com/gin-gonic/gin"
)

func currentUserID(c *gin.Context) uint64 {
	return middleware.CurrentUserID(c)
}

func parseUintParam(c *gin.Context, key string) (uint64, error) {
	value := c.Param(key)
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, apperror.BadRequest("参数不正确", err)
	}
	return parsed, nil
}
