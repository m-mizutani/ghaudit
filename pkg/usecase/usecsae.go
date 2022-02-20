package usecase

import (
	"fmt"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/infra"
	"github.com/m-mizutani/ghaudit/pkg/utils"
	"github.com/m-mizutani/goerr"
)

type Usecase struct {
	clients *infra.Clients
}

func New(clients *infra.Clients) *Usecase {
	return &Usecase{
		clients: clients,
	}
}

type regoInput struct {
	Repo          *github.Repository `json:"repo"`
	Branches      []*github.Branch   `json:"branches"`
	Collaborators []*github.User     `json:"collaborators"`
	Hooks         []*github.Hook     `json:"hooks"`
	Teams         []*github.Team     `json:"teams"`
	Timestamp     int64              `json:"timestamp"`
}

type regoOutput struct {
	Fail []*regoFail `json:"fail"`
}

type regoFail struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

type auditResult struct {
	regoFail
	RepoFullName string    `json:"repo"`
	ScannedAt    time.Time `json:"scanned_at"`
}

func (x *Usecase) Audit(ctx *types.Context, owner string) error {
	repos, err := x.clients.GitHubApp().GetRepos(ctx, owner)
	if err != nil {
		return goerr.Wrap(err).With("owner", owner)
	}
	utils.Logger.With("total repos", len(repos)).Debug("retried repository list")

	var results []*auditResult
	for _, repo := range repos {
		now := time.Now().UTC()
		repoName := repo.GetName()

		utils.Logger.With("repo", repoName).Debug("retrieving repository data")

		branches, err := x.clients.GitHubApp().GetBranches(ctx, owner, repoName)
		if err != nil {
			return goerr.Wrap(err).With("owner", owner).With("repo", repoName)
		}

		collaborators, err := x.clients.GitHubApp().GetCollaborators(ctx, owner, repoName)
		if err != nil {
			return goerr.Wrap(err).With("owner", owner).With("repo", repoName)
		}

		hooks, err := x.clients.GitHubApp().GetHooks(ctx, owner, repoName)
		if err != nil {
			return goerr.Wrap(err).With("owner", owner).With("repo", repoName)
		}

		teams, err := x.clients.GitHubApp().GetTeams(ctx, owner, repoName)
		if err != nil {
			return goerr.Wrap(err).With("owner", owner).With("repo", repoName)
		}

		input := &regoInput{
			Repo:          repo,
			Branches:      branches,
			Collaborators: collaborators,
			Hooks:         hooks,
			Teams:         teams,
			Timestamp:     now.Unix(),
		}
		var output regoOutput
		utils.Logger.With("repo", repoName).Debug("evaluating repository data")
		if err := x.clients.Policy().Eval(ctx, input, &output); err != nil {
			return goerr.Wrap(err).With("owner", owner).With("repo", repoName)
		}
		utils.Logger.With("output", output).Debug("evaluated")

		for _, fail := range output.Fail {
			results = append(results, &auditResult{
				regoFail:     *fail,
				RepoFullName: repo.GetFullName(),
				ScannedAt:    now,
			})
		}
	}

	if len(results) > 0 {
		fmt.Printf("\n===== %d violation detected =====\n", len(results))
		for _, result := range results {
			fmt.Printf("- [%s] %s: %s\n", result.RepoFullName, result.Category, result.Message)
		}
		fmt.Printf("\n")
		return types.ErrViolationDetected
	} else {
		fmt.Printf("\n----- No violation detected -----\n\n")
	}

	return nil
}
