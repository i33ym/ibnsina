package ibnsina

import (
	"regexp"
	"slices"
	"unicode/utf8"
)

var (
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	// ComplexPasswordRX = regexp.MustCompile("")
	// ModeratePasswordRX = regexp.MustCompile("")
	// URLRX = regexp.MustCompile("^(http|https)://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/[a-zA-Z0-9-._~:?#@!$&'()*+,;=]*)*$")
	// UsernameRX = regexp.MustCompile("")
)

type Validator struct {
	FieldErrors    map[string]string
	NonFieldErrors []string
}

func NewValidator() *Validator {
	return &Validator{
		FieldErrors:    make(map[string]string),
		NonFieldErrors: make([]string, 0),
	}
}

func (validator *Validator) Ok() bool {
	return len(validator.FieldErrors) == 0 && len(validator.NonFieldErrors) == 0
}

// func (validator *Validator) Clear() {
// 	validator = NewValidator()
// }

func (validator *Validator) Check(cond bool, key string, message string) {
	if !cond {
		validator.AddFieldError(key, message)
	}
}

func (validator *Validator) AddFieldError(key string, message string) {
	if _, exists := validator.FieldErrors[key]; !exists {
		validator.FieldErrors[key] = message
	}
}

func (validator *Validator) AddNonFieldError(message string) {
	if !slices.Contains(validator.NonFieldErrors, message) {
		validator.NonFieldErrors = append(validator.NonFieldErrors, message)
	}
}

func In[T comparable](value T, values []T) bool {
	return slices.Contains(values, value)
}

func MaxRunes(value string, maxLimit int) bool {
	return utf8.RuneCountInString(value) <= maxLimit
}

func MinRunes(value string, minLimit int) bool {
	return utf8.RuneCountInString(value) >= minLimit
}

func RunesInRange(value string, minLimit, maxLimit int) bool {
	return MinRunes(value, minLimit) && MaxRunes(value, maxLimit)
}

func MaxChars(value string, maxLimit int) bool {
	return len([]byte(value)) <= maxLimit
}

func MinChars(value string, minLimit int) bool {
	return len([]byte(value)) >= minLimit
}

func CharsInRange(value string, minLimit, maxLimit int) bool {
	return MinChars(value, minLimit) && MaxChars(value, maxLimit)
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func Uniques[T comparable](values []T) bool {
	uniques := make(map[T]bool)

	for index := 0; index < len(values); index++ {
		uniques[values[index]] = true
	}

	return len(uniques) == len(values)
}
