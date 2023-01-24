package helpers

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type FlagBinder struct {
	Viper   *viper.Viper
	FlagSet *pflag.FlagSet
}

func (f *FlagBinder) bind(key, name string) {
	err := f.Viper.BindPFlag(key, f.FlagSet.Lookup(name))
	if err != nil {
		panic(err)
	}
}

func (f *FlagBinder) BindBool(key string, value bool, usage string) {
	name := strings.ReplaceAll(key, ".", "-")
	f.FlagSet.Bool(name, value, usage)
	f.bind(key, name)
}

func (f *FlagBinder) BindString(key string, value string, usage string) {
	name := strings.ReplaceAll(key, ".", "-")
	f.FlagSet.String(name, value, usage)
	f.bind(key, name)
}

func (f *FlagBinder) BindStringSlice(key string, value []string, usage string) {
	name := strings.ReplaceAll(key, ".", "-")
	f.FlagSet.StringSlice(name, value, usage)
	f.bind(key, name)
}
