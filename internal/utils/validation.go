package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validatorInstance  = validator.New()
	usernameRegex      = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	noOuterSpacesRegex = regexp.MustCompile(`^\S.*\S$|^\S+$`)
)

func customUsername(fl validator.FieldLevel) bool {
	return usernameRegex.MatchString(fl.Field().String())
}

func customNoOuterSpaces(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if len(val) > 0 {
		return noOuterSpacesRegex.MatchString(val)
	}
	return true
}

func init() {
	validatorInstance.RegisterValidation("customUsername", customUsername)
	validatorInstance.RegisterValidation("customNoOuterSpaces", customNoOuterSpaces)
}

func ValidateStruct(s any) error {
	if err := validatorInstance.Struct(s); err != nil {
		errs := []string{}
		for _, err := range err.(validator.ValidationErrors) {
			errs = append(errs, fmt.Sprintf("%s: vaiolation in constarint '%s'", err.Field(), err.Tag()))
		}
		return errors.New(strings.Join(errs, ";"))
	}
	return nil
}
