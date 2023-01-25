package internal

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
	Configuration zap.Config
)

func AttachLoggingOptions(flagSet *pflag.FlagSet, viperInstance *viper.Viper) {

	flagSet.String("log-level", "debug", "Logging level of stdout (debug, info or error)")
	flagSet.String("log-env", "prod", "Logging env (prod or dev)")

	viperInstance.BindPFlag("log.level", flagSet.Lookup("log-level"))
	viperInstance.BindPFlag("log.env", flagSet.Lookup("log-env"))

	Configuration = zap.NewProductionConfig()

}

func LoadLogger(viperInstance *viper.Viper) {

	if viperInstance.GetString("log.env") == "dev" {
		Configuration = zap.NewDevelopmentConfig()
	}

	switch viperInstance.Get("log.level") {

	case "error":
		Configuration.Level.SetLevel(zap.ErrorLevel)

	case "info":
		Configuration.Level.SetLevel(zap.InfoLevel)

	default:
		Configuration.Level.SetLevel(zap.DebugLevel)

	}

	Configuration.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	logger, err := Configuration.Build()
	if err != nil {
		panic(err)
	}

	Logger = logger
	SugaredLogger = logger.Sugar()
}

func FlushLogger() {

	Logger.Sync()

}
