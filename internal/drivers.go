package internal

import (
	"context"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Driver interface {
	Bind(cfg *viper.Viper, flagSet *pflag.FlagSet)
	Load(ctx context.Context) error
}
