package parameter

import (
	"context"
	"io"
)

type ParameterPathProvider interface {
	GetParameterPath() string
}

type ParameterStore interface {
	Load(ctx context.Context, pathProvider ParameterPathProvider, parameterSet *ParameterSet) error
}

type ParameterFileDownloader interface {
	Download(ctx context.Context, source string, writer io.Writer) error
}
