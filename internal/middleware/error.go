package middleware

import (
	"errors"

	"baby-record-server/internal/apperror"
	"baby-record-server/internal/response"
	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			response.Failure(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
			return
		}

		response.Failure(c, 500, apperror.CodeInternal, "服务器开小差了")
	}
}
