package githubapp

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/go-github/v42/github"
	"github.com/m-mizutani/ghaudit/pkg/domain/model"
	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/goerr"
)

type loaderClient struct {
	input map[string]*model.RegoInput
}

func NewloaderClient(dir string) (*loaderClient, error) {
	client := &loaderClient{
		input: make(map[string]*model.RegoInput),
	}
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return goerr.Wrap(err)
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}

		fd, err := os.Open(path)
		if err != nil {
			return goerr.Wrap(err)
		}
		defer fd.Close()

		input := model.RegoInput{}
		if err := json.NewDecoder(fd).Decode(&input); err != nil {
			return goerr.Wrap(err)
		}

		client.input[input.Repo.GetFullName()] = &input

		return nil
	}); err != nil {
		return nil, goerr.Wrap(err)
	}
	return client, nil
}

func (x *loaderClient) GetRepos(ctx *types.Context, owner string) ([]*github.Repository, error) {
	var repos []*github.Repository
	for _, v := range x.input {
		repos = append(repos, v.Repo)
	}
	return repos, nil
}

func (x *loaderClient) GetBranches(ctx *types.Context, owner string, repo string) ([]*github.Branch, error) {
	return x.input[owner+"/"+repo].Branches, nil
}

func (x *loaderClient) GetCollaborators(ctx *types.Context, owner string, repo string) ([]*github.User, error) {
	return x.input[owner+"/"+repo].Collaborators, nil
}

func (x *loaderClient) GetHooks(ctx *types.Context, owner string, repo string) ([]*github.Hook, error) {
	return x.input[owner+"/"+repo].Hooks, nil
}

func (x *loaderClient) GetTeams(ctx *types.Context, owner string, repo string) ([]*github.Team, error) {
	return x.input[owner+"/"+repo].Teams, nil
}
