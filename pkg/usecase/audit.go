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
	"github.com/m-mizutani/ghaudit/pkg/infra/githubapp"
	"github.com/m-mizutani/ghaudit/pkg/utils"
	"github.com/m-mizutani/goerr"
)

type auditRecord struct {
	model.RegoFail
	Repo      *github.Repository
	ScannedAt time.Time
}

func createRegoInput(ctx *types.Context, client githubapp.Client, repo *github.Repository) (*model.RegoInput, error) {
	now := time.Now().UTC()
	repoName := repo.GetName()
	ownerName := repo.Owner.GetLogin()

	utils.Logger.With("repo", repoName).Trace("retrieving repository data")

	githubBranches, err := client.GetBranches(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err)
	}

	var branches []*model.RegoInputBranch
	for _, branch := range githubBranches {
		b := &model.RegoInputBranch{
			Branch: *branch,
		}
		if branch.GetProtected() {
			protection, err := client.GetBranchProtection(ctx, ownerName, repoName, branch.GetName())
			if err != nil {
				return nil, goerr.Wrap(err)
			}
			b.Protection = protection
		}
		branches = append(branches, b)
	}

	collaborators, err := client.GetCollaborators(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err)
	}

	hooks, err := client.GetHooks(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err)
	}

	teams, err := client.GetTeams(ctx, ownerName, repoName)
	if err != nil {
		return nil, goerr.Wrap(err)
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

func (x *Usecase) evaluate(ctx *types.Context, input *model.RegoInput) ([]*auditRecord, error) {
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

	var results []*auditRecord
	var output model.RegoOutput
	repoName := input.Repo.GetFullName()
	utils.Logger.With("repo", repoName).Trace("evaluating repository data")
	if err := x.clients.Policy().Query(ctx, input, &output); err != nil {
		return nil, goerr.Wrap(err).With("owner", input.Repo.Owner.GetLogin()).With("repo", repoName)
	}

	for _, fail := range output.Fail {
		results = append(results, &auditRecord{
			RegoFail: *fail,
			Repo:     input.Repo,
		})
	}

	return results, nil
}

func (x *Usecase) Audit(ctx *types.Context, owner string) error {
	startedAt := time.Now()

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

	result := newAuditResult(repos, startedAt)

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
			records, err := x.evaluate(ctx, input)
			if err != nil {
				return err
			}
			result.Add(records...)

		case err := <-errCh:
			if err != nil {
				return err
			}
		}
	}

	result.CompletedAt = time.Now()
	if err := x.output(ctx, result); err != nil {
		return err
	}

	return nil
}
