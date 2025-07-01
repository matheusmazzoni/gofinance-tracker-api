package validator

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
)

// Validatable defines an interface for any struct that wants to have its own
// custom struct-level validation logic.
type Validatable interface {
	Validate(sl validator.StructLevel)
}

// genericStructLevelValidation is the single validation function we register with the validator.
// It checks if the struct being validated implements our `Validatable` interface.
func genericStructLevelValidation(sl validator.StructLevel) {
	// Tries to assert the struct to our interface.
	if validatable, ok := sl.Current().Addr().Interface().(Validatable); ok {
		// If it does, we call its own Validate method.
		validatable.Validate(sl)
	}
}

// Init initializes our custom validator and registers all necessary logic with Gin.
// This should be called once when the application starts.
func Init() {
	// Get Gin's default validator engine.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {

		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
		// Register our generic struct-level validation function for each DTO
		// that implements the Validatable interface.
		v.RegisterStructValidation(genericStructLevelValidation,
			dto.AccountRequest{},
		)
	}
}
