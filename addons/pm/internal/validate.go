package internal

import "github.com/go-playground/validator/v10"

var v = validator.New(validator.WithRequiredStructEnabled())
