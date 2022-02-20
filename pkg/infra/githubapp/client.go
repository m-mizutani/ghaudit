package githubapp

import (
	"io"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v42/github"
	"github.com/m-mizutani/goerr"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
)

type Client interface {
	GetRepos(ctx *types.Context, owner string) ([]*github.Repository, error)
	GetBranches(ctx *types.Context, owner, repo string) ([]*github.Branch, error)
	GetCollaborators(ctx *types.Context, owner, repo string) ([]*github.User, error)
	GetHooks(ctx *types.Context, owner, repo string) ([]*github.Hook, error)
	GetTeams(ctx *types.Context, owner, repo string) ([]*github.Team, error)
}

type client struct {
	client *github.Client
}

func New(appID, installID int64, privateKey []byte) (Client, error) {
	itr, err := ghinstallation.New(http.DefaultTransport, appID, installID, privateKey)
	if err != nil {
		return nil, goerr.Wrap(err)
	}

	return &client{
		client: github.NewClient(&http.Client{Transport: itr}),
	}, nil
}

func (x *client) GetRepos(ctx *types.Context, owner string) ([]*github.Repository, error) {
	const perPage = 100
	page := 1
	var repos []*github.Repository

	for {
		got, resp, err := x.client.Repositories.List(ctx, owner, &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, goerr.Wrap(err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, types.ErrUnexpectedGitHubResp.New().
				With("code", resp.StatusCode).With("body", body)
		}

		repos = append(repos, got...)
		if len(got) < perPage {
			break
		}
	}

	return repos, nil
}

func (x *client) GetBranches(ctx *types.Context, owner, repo string) ([]*github.Branch, error) {
	const perPage = 100
	page := 1
	var branches []*github.Branch

	for {
		got, resp, err := x.client.Repositories.ListBranches(ctx, owner, repo, &github.BranchListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, goerr.Wrap(err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, types.ErrUnexpectedGitHubResp.New().
				With("code", resp.StatusCode).With("body", body)
		}

		branches = append(branches, got...)
		if len(got) < perPage {
			break
		}
	}

	return branches, nil
}

func (x *client) GetCollaborators(ctx *types.Context, owner, repo string) ([]*github.User, error) {
	const perPage = 100
	page := 1
	var users []*github.User

	for {
		got, resp, err := x.client.Repositories.ListCollaborators(ctx, owner, repo, &github.ListCollaboratorsOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, goerr.Wrap(err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, types.ErrUnexpectedGitHubResp.New().
				With("code", resp.StatusCode).With("body", body)
		}

		users = append(users, got...)
		if len(got) < perPage {
			break
		}
	}

	return users, nil
}

func (x *client) GetHooks(ctx *types.Context, owner, repo string) ([]*github.Hook, error) {
	const perPage = 100
	page := 1
	var hooks []*github.Hook

	for {
		got, resp, err := x.client.Repositories.ListHooks(ctx, owner, repo, &github.ListOptions{
			Page:    page,
			PerPage: perPage,
		})
		if err != nil {
			return nil, goerr.Wrap(err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, types.ErrUnexpectedGitHubResp.New().
				With("code", resp.StatusCode).With("body", body)
		}

		hooks = append(hooks, got...)
		if len(got) < perPage {
			break
		}
	}

	return hooks, nil
}

func (x *client) GetTeams(ctx *types.Context, owner, repo string) ([]*github.Team, error) {
	const perPage = 100
	page := 1
	var teams []*github.Team

	for {
		got, resp, err := x.client.Repositories.ListTeams(ctx, owner, repo, &github.ListOptions{
			Page:    page,
			PerPage: perPage,
		})
		if err != nil {
			return nil, goerr.Wrap(err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, types.ErrUnexpectedGitHubResp.New().
				With("code", resp.StatusCode).With("body", body)
		}

		teams = append(teams, got...)
		if len(got) < perPage {
			break
		}
	}

	return teams, nil
}
