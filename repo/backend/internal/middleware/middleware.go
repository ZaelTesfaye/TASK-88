package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErrors "backend/internal/errors"
	"backend/internal/logging"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)
		c.Next()
	}
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return logging.RequestLoggerMiddleware()
}

func EgressGuardMiddleware(allowedHosts []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !isAllowedHost(clientIP, allowedHosts) {
			logging.Warn("middleware", "egress_guard", "blocked request from non-LAN IP",
			)
			appErrors.RespondWithError(c, http.StatusForbidden, appErrors.Forbidden(
				fmt.Sprintf("access denied: client IP %s is not in allowed network range", clientIP),
			))
			return
		}
		c.Next()
	}
}

func isAllowedHost(clientIP string, allowedHosts []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, host := range allowedHosts {
		if strings.Contains(host, "/") {
			_, cidr, err := net.ParseCIDR(host)
			if err != nil {
				continue
			}
			if cidr.Contains(ip) {
				return true
			}
		} else {
			if clientIP == host {
				return true
			}
			if host == "localhost" && (clientIP == "127.0.0.1" || clientIP == "::1") {
				return true
			}
		}
	}
	return false
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("middleware", "recovery", fmt.Sprintf("panic recovered: %v", r))

				correlationID, _ := c.Get("correlation_id")
				cid, _ := correlationID.(string)

				c.AbortWithStatusJSON(http.StatusInternalServerError, &appErrors.AppError{
					Code:          "INTERNAL_ERROR",
					Message:       "An unexpected error occurred",
					CorrelationID: cid,
				})
			}
		}()
		c.Next()
	}
}
