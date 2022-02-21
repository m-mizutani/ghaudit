package usecase

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/m-mizutani/ghaudit/pkg/domain/model"
	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/infra"
	"github.com/m-mizutani/ghaudit/pkg/infra/githubapp"
	"github.com/m-mizutani/ghaudit/pkg/utils"
	"github.com/m-mizutani/goerr"
)

type Usecase struct {
	clients *infra.Clients
	thread  int64
	limit   int64
	dumpDir string
}

func New(clients *infra.Clients, options ...Option) *Usecase {
	uc := &Usecase{
		clients: clients,
		thread:  4,
	}

	for _, opt := range options {
		opt(uc)
	}

	return uc
}

type Option func(uc *Usecase)

func WithThread(n int64) Option {
	return func(uc *Usecase) {
		uc.thread = n
	}
}

func WithLimit(n int64) Option {
	return func(uc *Usecase) {
		uc.limit = n
	}
}

func WithDump(dir string) Option {
	return func(uc *Usecase) {
		uc.dumpDir = filepath.Clean(dir)
	}
}

type auditResult struct {
	model.RegoFail
	RepoFullName string    `json:"repo"`
	ScannedAt    time.Time `json:"scanned_at"`
}

func createRegoInput(ctx *types.Context, client githubapp.Client, repo *github.Repository) (*model.RegoInput, error) {
	now := time.Now().UTC()
	repoName := repo.GetName()
	ownerName := repo.Owner.GetLogin()

	utils.Logger.With("repo", repoName).Trace("retrieving repository data")

	branches, err := client.GetBranches(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err).With("owner", ownerName).With("repo", repoName)
	}

	collaborators, err := client.GetCollaborators(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err).With("owner", ownerName).With("repo", repoName)
	}

	hooks, err := client.GetHooks(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err).With("owner", ownerName).With("repo", repoName)
	}

	teams, err := client.GetTeams(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err).With("owner", ownerName).With("repo", repoName)
	}

	input := &model.RegoInput{
		Repo:          repo,
		Branches:      branches,
		Collaborators: collaborators,
		Hooks:         hooks,
		Teams:         teams,
		Timestamp:     now.Unix(),
	}

	utils.Logger.With("repo", repoName).Trace("created input")

	return input, nil
}

func (x *Usecase) evaluate(ctx *types.Context, input *model.RegoInput) ([]*auditResult, error) {
	if x.dumpDir != "" {
		path := filepath.Join(x.dumpDir, fmt.Sprintf("%s.json", input.Repo.GetName()))
		fd, err := os.Create(path)
		if err != nil {
			return nil, goerr.Wrap(err)
		}
		if err := json.NewEncoder(fd).Encode(input); err != nil {
			return nil, goerr.Wrap(err)
		}
	}

	var results []*auditResult
	var output model.RegoOutput
	repoName := input.Repo.GetFullName()
	utils.Logger.With("repo", repoName).Trace("evaluating repository data")
	if err := x.clients.Policy().Eval(ctx, input, &output); err != nil {
		return nil, goerr.Wrap(err).With("owner", input.Repo.Owner.GetLogin()).With("repo", repoName)
	}

	for _, fail := range output.Fail {
		results = append(results, &auditResult{
			RegoFail:     *fail,
			RepoFullName: input.Repo.GetFullName(),
			ScannedAt:    time.Unix(input.Timestamp, 0),
		})
	}

	return results, nil
}

func (x *Usecase) Audit(ctx *types.Context, owner string) error {
	if x.dumpDir != "" {
		if err := os.MkdirAll(x.dumpDir, 0777); err != nil {
			return goerr.Wrap(err)
		}
	}

	repos, err := x.clients.GitHubApp().GetRepos(ctx, owner)
	if err != nil {
		return goerr.Wrap(err).With("owner", owner)
	}
	utils.Logger.With("total repos", len(repos)).Trace("retried repository list")

	limit := len(repos)
	if 0 < x.limit && int(x.limit) < limit {
		limit = int(x.limit)
	}

	var allResults []*auditResult

	errCh := make(chan error)
	inputCh := make(chan *model.RegoInput, limit)
	repoCh := make(chan *github.Repository, limit)

	var wg sync.WaitGroup

	for i := 0; i < int(x.thread); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoCh {
				input, err := createRegoInput(ctx, x.clients.GitHubApp(), repo)
				if err != nil {
					errCh <- err
					return
				}
				inputCh <- input
			}
		}()
	}
	go func() {
		wg.Wait()
		close(inputCh)
	}()

	for i := 0; i < limit; i++ {
		repoCh <- repos[i]
	}
	close(repoCh)
	utils.Logger.With("limit", limit).Trace("pushed repos")

Loop:
	for {
		select {
		case input := <-inputCh:
			if input == nil {
				break Loop
			}
			utils.Logger.With("repo", input.Repo.GetFullName()).Info("retrieved repo data")
			results, err := x.evaluate(ctx, input)
			if err != nil {
				return err
			}
			allResults = append(allResults, results...)

		case err := <-errCh:
			if err != nil {
				return err
			}
		}
	}

	if len(allResults) > 0 {
		fmt.Printf("\n===== %d violation detected =====\n", len(allResults))
		for _, result := range allResults {
			fmt.Printf("- [%s] %s: %s\n", result.RepoFullName, result.Category, result.Message)
		}
		fmt.Printf("\n")
		return types.ErrViolationDetected
	} else {
		fmt.Printf("\n----- No violation detected -----\n\n")
	}

	return nil
}
