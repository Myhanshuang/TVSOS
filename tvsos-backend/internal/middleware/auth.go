// Package middleware 提供 HTTP 中间件。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/handler"
)

// AuthRequired 简易 token 鉴权（演示级）：
// 支持 Authorization: Bearer <token> 与 ?token=<token>（WS 握手无法自定义头，用 query）。
// 校验通过后在 context 写入 userId 与 authToken。
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if header := c.GetHeader("Authorization"); header != "" {
			token = strings.TrimPrefix(header, "Bearer ")
		}
		if token == "" {
			token = c.Query("token")
		}
		userID := handler.ResolveToken(token)
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 2, "message": "未登录或登录已过期"})
			return
		}
		c.Set("userId", userID)
		c.Set("authToken", token)
		c.Next()
	}
}
