package githubapp_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/infra/githubapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubClient(t *testing.T) {
	envAppID := os.Getenv(types.EnvAppID)
	envInstallID := os.Getenv(types.EnvInstallID)
	envKeyFile := os.Getenv(types.EnvPrivateKeyFile)

	if envAppID == "" || envInstallID == "" || envKeyFile == "" {
		t.Skip("environment variables required")
	}

	appID, err := strconv.ParseInt(envAppID, 10, 64)
	require.NoError(t, err)
	installID, err := strconv.ParseInt(envInstallID, 10, 64)
	require.NoError(t, err)
	keyFile := filepath.Join("..", "..", "..", filepath.Clean(envKeyFile))
	keyData, err := os.ReadFile(keyFile)
	require.NoError(t, err)

	client, err := githubapp.New(appID, installID, keyData)
	require.NoError(t, err)

	repos, err := client.GetRepos(types.NewContext(), "mizutani-sandbox")
	require.NoError(t, err)
	require.Len(t, repos, 2)
	repoNames := []string{repos[0].GetName(), repos[1].GetName()}
	assert.Contains(t, repoNames, "test-repo")
}
