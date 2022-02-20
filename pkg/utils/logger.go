package utils

import (
	"os"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zlog"
	"github.com/m-mizutani/zlog/filter"
)

var Logger = zlog.New()

func RenewLogger(logLevel, logFormat string) error {
	var writer *zlog.Writer
	switch logFormat {
	case "text":
		writer = zlog.NewWriterWith(zlog.NewConsoleFormatter(), os.Stdout)
	case "json":
		writer = zlog.NewWriterWith(zlog.NewConsoleFormatter(), os.Stdout)
	default:
		return goerr.Wrap(types.ErrInvalidConfig, "invalid log format").With("format", logFormat)
	}
	Logger = zlog.New(
		zlog.WithEmitter(writer),
		zlog.WithLogLevel(logLevel),
		zlog.WithFilters(
			filter.Tag("secret"),
		),
	)

	return nil
}
