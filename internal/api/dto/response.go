package dto

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

// ErrorResponse is the standardized DTO for sending error messages.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

func BindAndValidate(c *gin.Context, obj any) bool {
	logger := zerolog.Ctx(c.Request.Context())
	if err := c.ShouldBindJSON(obj); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			formatErrors := formatValidationErrors(validationErrors)
			for _, formatError := range formatErrors {
				logger.Err(err).Msg(formatError)
			}
			SendError(c, http.StatusBadRequest, "Validation failed", formatErrors)
		} else {
			erroMsg := "Invalid request body"
			logger.Err(err).Msg(erroMsg)
			SendError(c, http.StatusBadRequest, erroMsg, nil)
		}
		return false
	}
	return true
}

// SendSuccessResponse sends a standardized successful HTTP response.
func SendSuccessResponse(c *gin.Context, code int, data any) {
	c.JSON(code, data)
}

// SendErrorResponse sends a standardized error HTTP response and aborts the request.
func SendErrorResponse(c *gin.Context, code int, message string) {
	response := ErrorResponse{
		Error: message,
	}
	c.AbortWithStatusJSON(code, response)
}

// TODO: Using this one instead of SendErrorResponse func
func SendError(c *gin.Context, code int, message string, details map[string]string) {
	response := ErrorResponse{
		Error:   message,
		Details: details,
	}
	c.AbortWithStatusJSON(code, response)
}

// formatValidationErrors is a private helper that translates validator errors
// into a simple map[string]string for the API response.
func formatValidationErrors(errs validator.ValidationErrors) map[string]string {
	fields := make(map[string]string)
	for _, fieldErr := range errs {
		// Uses the 'json' tag name we registered in the validator init.
		fieldName := fieldErr.Field()

		// You can customize messages for each validation tag.
		switch fieldErr.Tag() {
		case "required":
			fields[fieldName] = fmt.Sprintf("The '%s' field is required.", fieldName)
		case "min":
			fields[fieldName] = fmt.Sprintf("The '%s' field must be at least %s characters long.", fieldName, fieldErr.Param())
		case "max":
			fields[fieldName] = fmt.Sprintf("The '%s' field must not exceed %s characters.", fieldName, fieldErr.Param())
		case "oneof":
			fields[fieldName] = fmt.Sprintf("The '%s' field must be one of: [%s].", fieldName, fieldErr.Param())

		// Custom tags from the struct-level validation
		case "required_if_credit_card", "required_for_credit_card":
			fields[fieldName] = fmt.Sprintf("The '%s' field is required for credit card accounts.", fieldName)
		case "day":
			fields[fieldName] = fmt.Sprintf("The '%s' field must be between 1 and 31. '%s' values is not a valid.", fieldName, fieldErr.Param())
		case "gt_credit_card":
			fields[fieldName] = fmt.Sprintf("The '%s' field must be greater than %s for credit card accounts.", fieldName, fieldErr.Param())
		case "ge_credit_card":
			fields[fieldName] = fmt.Sprintf("The '%s' field must be greater than or equal to %s for credit card accounts.", fieldName, fieldErr.Param())
		case "le_credit_card":
			fields[fieldName] = fmt.Sprintf("The '%s' field must be less than or equal to %s for credit card accounts.", fieldName, fieldErr.Param())
		case "not_allowed_for_non_credit_card":
			fields[fieldName] = fmt.Sprintf("The '%s' field is not allowed for this account type.", fieldName)

		default:
			fields[fieldName] = fmt.Sprintf("The '%s' field has an invalid value.", fieldName)
		}
	}
	return fields
}
