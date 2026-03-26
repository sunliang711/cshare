package logger

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(New),
)

func New() zerolog.Logger {
	zerolog.CallerMarshalFunc = shortCaller

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

func shortCaller(_ uintptr, file string, line int) string {
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		file = strings.Join(parts[len(parts)-2:], "/")
	}
	return file + ":" + strconv.Itoa(line)
}
