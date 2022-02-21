package cmd

import (
	"errors"
	"os"

	"github.com/m-mizutani/ghaudit/pkg/domain/model"
	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/infra"
	"github.com/m-mizutani/ghaudit/pkg/infra/githubapp"
	"github.com/m-mizutani/ghaudit/pkg/infra/policy"
	"github.com/m-mizutani/ghaudit/pkg/usecase"
	"github.com/m-mizutani/ghaudit/pkg/utils"
	"github.com/m-mizutani/goerr"

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
				Required:    true,
				Destination: &cfg.Owner,
			},

			// GitHub App
			&cli.Int64Flag{
				Name:        "app-id",
				EnvVars:     []string{types.EnvAppID},
				Usage:       "GitHub App ID",
				Required:    true,
				Destination: &cfg.AppID,
			},
			&cli.Int64Flag{
				Name:        "install-id",
				EnvVars:     []string{types.EnvInstallID},
				Usage:       "GitHub Install ID",
				Required:    true,
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
				Value:       4,
			},
			&cli.Int64Flag{
				Name:        "limit",
				Usage:       "Limit of auditing repository",
				EnvVars:     []string{types.EnvLimit},
				Destination: &cfg.Limit,
				Value:       0,
			},
		},
		Before: func(c *cli.Context) error {
			cfg.Headers = headers.Value()

			if err := cfg.Validate(); err != nil {
				return err
			}

			if err := utils.RenewLogger(cfg.LogLevel, cfg.LogFormat); err != nil {
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
			utils.Logger.Err(err).Error("exit with error")
			return goerr.Wrap(err)
		}
	}

	return nil
}

func action(cfg *model.Config) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		utils.Logger.With("config", cfg).Info("Audit starting...")

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

		ghapp, err := githubapp.New(cfg.AppID, cfg.InstallID, privateKey)
		if err != nil {
			return goerr.Wrap(err).With("appID", cfg.AppID).With("installID", cfg.InstallID)
		}

		var policyClient policy.Client
		if cfg.Policy != "" {
			utils.Logger.With("policy", cfg.Policy).Info("Use local policy file(s)")
			p, err := policy.NewLocal(cfg.Policy, policy.WithPackage(cfg.Package))
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
			p, err := policy.NewRemoteWithHTTPClient(cfg.URL, httpClient)
			if err != nil {
				return err
			}
			policyClient = p
		}

		clients := infra.New(
			infra.WithGitHubApp(ghapp),
			infra.WithPolicy(policyClient),
		)
		uc := usecase.New(clients,
			usecase.WithLimit(cfg.Limit),
			usecase.WithThread(cfg.Thread),
		)

		ctx := types.NewContext(types.WithCtx(c.Context))
		if err := uc.Audit(ctx, cfg.Owner); err != nil {
			return err
		}

		return nil

	}
}
