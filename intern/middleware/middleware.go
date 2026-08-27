package auth

import (
	"bank/intern/metrics"
	"bank/intern/service"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtService *service.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token format"})
			return
		}
		tokenString := parts[1]
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
			return
		}
		log.Println(" Claims from token:", claims.UserID, claims.Login, claims.Role)
		c.Set("user_id", claims.UserID)
		c.Set("login", claims.Login)
		c.Set("role", claims.Role)
		c.Next()
	}
}
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleRaw, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		role, ok := roleRaw.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid role type"})
			return
		}

		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Next()
	}
}
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Увеличиваем счётчик активных запросов
		metrics.HttpActiveRequests.Inc()
		defer metrics.HttpActiveRequests.Dec()

		// Выполняем запрос
		c.Next()

		// Собираем метрики после выполнения
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()
		statusStr := strconv.Itoa(status)

		metrics.HttpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
		metrics.HttpRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}
