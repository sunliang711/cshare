package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(New),
)

func New() zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	return zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()
}
