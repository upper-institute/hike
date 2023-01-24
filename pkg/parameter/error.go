package parameter

import "errors"

var (
	SeparatorNotFoundErr    = errors.New("Unable find separator to load parameter")
	InvalidParameterTypeErr = errors.New("Invalid parameter type")
	FileNotFoundErr         = errors.New("File not found for this key")
	LoadOnlyFileTypeErr     = errors.New("Load method applies only for parameter type 'file'")
	UnknownSchemeErr        = errors.New("Unknown parameter scheme")
)
