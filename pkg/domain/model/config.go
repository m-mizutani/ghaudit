package model

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/goerr"
)

type Config struct {
	Owner string

	AppID          int64
	InstallID      int64
	PrivateKeyFile string
	PrivateKeyData string `zlog:"secret"`

	Policy  string
	Package string

	URL     string
	Headers []string `zlog:"secret"`

	LogFormat    string
	LogLevel     string
	SlackWebhook string `zlog:"secret"`
	Fail         bool

	Thread  int64
	Limit   int64
	DumpDir string
}

func (x *Config) Validate() error {
	if err := validation.ValidateStruct(x,
		validation.Field(&x.Owner, validation.Required, validation.Match(regexp.MustCompile(`[a-zA-Z0-9-_]+`))),
		validation.Field(&x.AppID, validation.Required),
		validation.Field(&x.InstallID, validation.Required),
		validation.Field(&x.LogFormat, validation.In("text", "json"), validation.Required),
		validation.Field(&x.LogLevel, validation.In("trace", "debug", "info", "warn", "error"), validation.Required),
		validation.Field(&x.URL, is.URL),
	); err != nil {
		return types.ErrInvalidConfig.Wrap(err)
	}

	if (x.PrivateKeyFile == "" && x.PrivateKeyData == "") ||
		(x.PrivateKeyFile != "" && x.PrivateKeyData != "") {
		return goerr.Wrap(types.ErrInvalidConfig, "either one of private key file or data is required")
	}

	if x.Policy == "" && x.URL == "" {
		return goerr.Wrap(types.ErrInvalidConfig, "either one of policy dir or opa server URL is required")
	}

	return nil
}
