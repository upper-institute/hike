package parameter

import "errors"

var (
	SeparatorNotFoundErr    = errors.New("Unable find separator to load parameter")
	InvalidParameterTypeErr = errors.New("Invalid parameter type")
	FileNotFoundErr         = errors.New("File not found for this key (maybe it's an env?)")
)
