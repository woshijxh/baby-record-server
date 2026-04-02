package middleware

import (
	"strings"

	"baby-record-server/internal/apperror"
	"baby-record-server/internal/auth"
	"github.com/gin-gonic/gin"
)

const currentUserIDKey = "currentUserID"

func Auth(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Error(apperror.Unauthorized("缺少认证信息", nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		tokenString = strings.TrimSpace(tokenString)
		if tokenString == "" {
			c.Error(apperror.Unauthorized("token 不能为空", nil))
			c.Abort()
			return
		}

		claims, err := manager.Parse(tokenString)
		if err != nil {
			c.Error(apperror.Unauthorized("token 无效或已过期", err))
			c.Abort()
			return
		}

		c.Set(currentUserIDKey, claims.UserID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) uint64 {
	value, _ := c.Get(currentUserIDKey)
	userID, _ := value.(uint64)
	return userID
}
