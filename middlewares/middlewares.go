package middlewares

import (
	"account-management/utils.go"

	"github.com/gin-gonic/gin"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := utils.TokenValid(c)
		if err != nil {
			c.JSON(500, gin.H{
				"messages": "Unauthorized",
				"error":    err.Error(),
				"status":   500,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
