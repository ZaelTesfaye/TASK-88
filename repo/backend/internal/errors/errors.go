package errors

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Code          string      `json:"code"`
	Message       string      `json:"message"`
	Details       interface{} `json:"details,omitempty"`
	CorrelationID string      `json:"correlationId"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewAppError(code, message string, details interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func BadRequest(message string, details interface{}) *AppError {
	return NewAppError("BAD_REQUEST", message, details)
}

func Unauthorized(message string) *AppError {
	return NewAppError("AUTH_REQUIRED", message, nil)
}

func Forbidden(message string) *AppError {
	return NewAppError("FORBIDDEN", message, nil)
}

func NotFound(message string) *AppError {
	return NewAppError("NOT_FOUND", message, nil)
}

func Conflict(message string, details interface{}) *AppError {
	return NewAppError("CONFLICT", message, details)
}

func ValidationError(message string, details interface{}) *AppError {
	return NewAppError("VALIDATION_ERROR", message, details)
}

func InternalError(message string) *AppError {
	return NewAppError("INTERNAL_ERROR", message, nil)
}

func RespondWithError(c *gin.Context, statusCode int, appErr *AppError) {
	correlationID, exists := c.Get("correlation_id")
	if exists {
		appErr.CorrelationID = correlationID.(string)
	}
	c.AbortWithStatusJSON(statusCode, appErr)
}

func RespondBadRequest(c *gin.Context, message string, details interface{}) {
	RespondWithError(c, http.StatusBadRequest, BadRequest(message, details))
}

func RespondUnauthorized(c *gin.Context, message string) {
	RespondWithError(c, http.StatusUnauthorized, Unauthorized(message))
}

func RespondForbidden(c *gin.Context, message string) {
	RespondWithError(c, http.StatusForbidden, Forbidden(message))
}

func RespondNotFound(c *gin.Context, message string) {
	RespondWithError(c, http.StatusNotFound, NotFound(message))
}

func RespondConflict(c *gin.Context, message string, details interface{}) {
	RespondWithError(c, http.StatusConflict, Conflict(message, details))
}

func RespondValidationError(c *gin.Context, message string, details interface{}) {
	RespondWithError(c, http.StatusUnprocessableEntity, ValidationError(message, details))
}

func RespondInternalError(c *gin.Context, message string) {
	RespondWithError(c, http.StatusInternalServerError, InternalError(message))
}

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		lastErr := c.Errors.Last()
		if lastErr == nil {
			return
		}

		if appErr, ok := lastErr.Err.(*AppError); ok {
			correlationID, exists := c.Get("correlation_id")
			if exists {
				appErr.CorrelationID = correlationID.(string)
			}

			var statusCode int
			switch appErr.Code {
			case "BAD_REQUEST":
				statusCode = http.StatusBadRequest
			case "AUTH_REQUIRED":
				statusCode = http.StatusUnauthorized
			case "FORBIDDEN":
				statusCode = http.StatusForbidden
			case "NOT_FOUND":
				statusCode = http.StatusNotFound
			case "CONFLICT":
				statusCode = http.StatusConflict
			case "VALIDATION_ERROR":
				statusCode = http.StatusUnprocessableEntity
			default:
				statusCode = http.StatusInternalServerError
			}

			c.JSON(statusCode, appErr)
			return
		}

		correlationID, _ := c.Get("correlation_id")
		cid, _ := correlationID.(string)
		c.JSON(http.StatusInternalServerError, &AppError{
			Code:          "INTERNAL_ERROR",
			Message:       "An unexpected error occurred",
			CorrelationID: cid,
		})
	}
}
