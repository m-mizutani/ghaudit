package cmd

import (
	"errors"
	"os"

	"github.com/m-mizutani/ghaudit/pkg/domain/model"
	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/infra"
	"github.com/m-mizutani/ghaudit/pkg/infra/githubapp"
	"github.com/m-mizutani/ghaudit/pkg/infra/notify"
	"github.com/m-mizutani/ghaudit/pkg/usecase"
	"github.com/m-mizutani/ghaudit/pkg/utils"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/opac"

	"github.com/urfave/cli/v2"
)

func Run(argv []string) error {
	cfg := &model.Config{}
	var headers cli.StringSlice
	app := &cli.App{
		Name:  "ghaudit",
		Usage: "GitHub Audit with OPA/Rego",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "owner",
				Aliases:     []string{"o"},
				Usage:       "GitHub owner (or organization) name to be audited",
				EnvVars:     []string{types.EnvOwner},
				Destination: &cfg.Owner,
			},

			// GitHub App
			&cli.Int64Flag{
				Name:        "app-id",
				EnvVars:     []string{types.EnvAppID},
				Usage:       "GitHub App ID",
				Destination: &cfg.AppID,
			},
			&cli.Int64Flag{
				Name:        "install-id",
				EnvVars:     []string{types.EnvInstallID},
				Usage:       "GitHub Install ID",
				Destination: &cfg.InstallID,
			},
			&cli.StringFlag{
				Name:        "private-key-file",
				EnvVars:     []string{types.EnvPrivateKeyFile},
				Usage:       "GitHub App private key file path",
				Destination: &cfg.PrivateKeyFile,
			},
			&cli.StringFlag{
				Name:        "private-key-data",
				EnvVars:     []string{types.EnvPrivateKeyData},
				Usage:       "GitHub App private key data path",
				Destination: &cfg.PrivateKeyData,
			},

			// OPA/Rego
			&cli.StringFlag{
				Name:        "policy",
				Aliases:     []string{"p"},
				EnvVars:     []string{types.EnvPolicy},
				Usage:       "Local Rego policy dir/file",
				Destination: &cfg.Policy,
			},
			&cli.StringFlag{
				Name:        "package",
				EnvVars:     []string{types.EnvPackage},
				Usage:       "Inquiry policy package name",
				Destination: &cfg.Package,
				Value:       "github.repo",
			},
			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				EnvVars:     []string{types.EnvURL},
				Usage:       "OPA server URL",
				Destination: &cfg.URL,
			},
			&cli.StringSliceFlag{
				Name:        "header",
				Aliases:     []string{"H"},
				EnvVars:     []string{types.EnvHeader},
				Usage:       "HTTP Header(s) of a request to OPA server",
				Destination: &headers,
			},

			// Misc options
			&cli.StringFlag{
				Name:        "log-level",
				Aliases:     []string{"l"},
				Usage:       "Log level [error|warn|info|debug|trace]",
				EnvVars:     []string{types.EnvLogLevel},
				Destination: &cfg.LogLevel,
				Value:       "info",
			},
			&cli.StringFlag{
				Name:        "log-format",
				Aliases:     []string{"f"},
				Usage:       "Log format [text|json]",
				EnvVars:     []string{types.EnvLogFormat},
				Destination: &cfg.LogFormat,
				Value:       "text",
			},
			&cli.BoolFlag{
				Name:        "fail",
				Usage:       "Exit with non-zero code when detecting violation",
				EnvVars:     []string{types.EnvFail},
				Destination: &cfg.Fail,
			},

			// Runtime options
			&cli.Int64Flag{
				Name:        "thread",
				Usage:       "Thread num",
				EnvVars:     []string{types.EnvThread},
				Destination: &cfg.Thread,
				Value:       1,
			},
			&cli.Int64Flag{
				Name:        "limit",
				Usage:       "Limit of auditing repository",
				EnvVars:     []string{types.EnvLimit},
				Destination: &cfg.Limit,
				Value:       0,
			},

			&cli.StringFlag{
				Name:        "dump",
				Usage:       "Directory to dump input data",
				EnvVars:     []string{types.EnvDumpDir},
				Destination: &cfg.DumpDir,
			},
			&cli.StringFlag{
				Name:        "load",
				Usage:       "Directory to load input data",
				EnvVars:     []string{types.EnvLoadDir},
				Destination: &cfg.LoadDir,
			},

			&cli.StringFlag{
				Name:        "slack-webhook",
				Usage:       "Slack incoming webhook URL to notify violation",
				EnvVars:     []string{types.EnvSlackWebhook},
				Destination: &cfg.SlackWebhook,
			},
		},
		Before: func(c *cli.Context) error {
			cfg.Headers = headers.Value()
			if err := utils.RenewLogger(cfg.LogLevel, cfg.LogFormat); err != nil {
				return err
			}
			utils.Logger.With("config", cfg).Info("Setting up...")

			if err := cfg.Validate(); err != nil {
				return err
			}

			return nil
		},

		Action: action(cfg),
	}

	if err := app.Run(argv); err != nil {
		if errors.Is(err, types.ErrViolationDetected) {
			if cfg.Fail {
				return err
			}
			// nothing to do ErrViolationDetected without cfg.Fail
		} else {
			log := utils.Logger.Log()
			var goErr *goerr.Error
			if errors.As(err, &goErr) {
				values := goErr.Values()
				if len(values) > 0 {
					for k, v := range goErr.Values() {
						log = log.With(k, v)
					}
				}

				if cfg.LogLevel == "debug" || cfg.LogLevel == "trace" {
					log = log.With("trace", goErr.Stacks())
				}
			}
			log.Error(err.Error())
			return goerr.Wrap(err)
		}
	}

	return nil
}

func action(cfg *model.Config) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		var ghapp githubapp.Client

		if cfg.LoadDir == "" {
			var privateKey []byte
			if cfg.PrivateKeyData != "" {
				privateKey = []byte(cfg.PrivateKeyData)
			} else {
				raw, err := os.ReadFile(cfg.PrivateKeyFile)
				if err != nil {
					return goerr.Wrap(err, "failed to read private key file")
				}
				privateKey = raw
			}

			app, err := githubapp.New(cfg.AppID, cfg.InstallID, privateKey)
			if err != nil {
				return goerr.Wrap(err).With("appID", cfg.AppID).With("installID", cfg.InstallID)
			}
			ghapp = app
		} else {
			loader, err := githubapp.NewloaderClient(cfg.LoadDir)
			if err != nil {
				return err
			}
			ghapp = loader
		}

		var policyClient opac.Client
		if cfg.Policy != "" {
			utils.Logger.With("policy", cfg.Policy).Info("Use local policy file(s)")
			p, err := opac.NewLocal(opac.WithDir(cfg.Policy), opac.WithPackage(cfg.Package))
			if err != nil {
				return err
			}
			policyClient = p
		} else if cfg.URL != "" {
			utils.Logger.With("url", cfg.URL).Info("Use local policy file(s)")
			httpClient, err := newHTTPClient(cfg.Headers)
			if err != nil {
				return err
			}
			p, err := opac.NewRemote(cfg.URL, opac.WithHTTPClient(httpClient))
			if err != nil {
				return err
			}
			policyClient = p
		}

		infraOptions := []infra.Option{
			infra.WithGitHubApp(ghapp),
			infra.WithPolicy(policyClient),
		}
		if cfg.SlackWebhook != "" {
			infraOptions = append(infraOptions, infra.WithSlack(notify.NewSlackWebhook(cfg.SlackWebhook)))
		}

		clients := infra.New(infraOptions...)

		ucOptions := []usecase.Option{
			usecase.WithLimit(cfg.Limit),
			usecase.WithThread(cfg.Thread),
		}
		if cfg.DumpDir != "" {
			ucOptions = append(ucOptions, usecase.WithDump(cfg.DumpDir))
		}

		uc := usecase.New(clients, ucOptions...)

		ctx := types.NewContext(types.WithCtx(c.Context))
		if err := uc.Audit(ctx, cfg.Owner); err != nil {
			return err
		}

		return nil

	}
}
