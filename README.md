# ghaudit [![Gosec](https://github.com/m-mizutani/ghaudit/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/ghaudit/actions/workflows/gosec.yml) [![Lint](https://github.com/m-mizutani/ghaudit/actions/workflows/lint.yml/badge.svg)](https://github.com/m-mizutani/ghaudit/actions/workflows/lint.yml) [![Test](https://github.com/m-mizutani/ghaudit/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/ghaudit/actions/workflows/test.yml)

CLI audit tool for GitHub repositories with [OPA/Rego](https://www.openpolicyagent.org/docs/latest/policy-language/).

![](https://user-images.githubusercontent.com/605953/155263397-5b0363df-6c24-4ad8-80c7-50fbe0c82176.png)

## Features

- Crawls GitHub repository meta data of your organization
- Evaluates the meta data with policy written by [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) or inquiry to [OPA](https://github.com/open-policy-agent/opa) server
- Exit with non-zero when detecting violation and notify the violation to Slack

## Setup

### 1) Create a new GitHub App

1. Go to https://github.com/organizations/{your_org_name}/settings/apps and click `New GitHub App`
2. Input required fields and grant following permissions. Then click `Create GitHub App`
    - Repository permissions
        - Administration: Read-only
        - Content: Read-only
        - Webhooks: Read-only
3. Create key by clicking `Generate a private key` and save it.
4. Move `Install App` page from left side bar and click `Install` button of the organization you want to install

Please note the following items

- AppID: You can find it in https://github.com/settings/apps/{your_app_name}
- InstallID: You can find it in installation page https://github.com/organizations/{your_org_name}/settings/installations/{Install ID}

### 2) Creating policy by Rego

#### Policy rules

- Package name: `github.repo`
- Input data
    - `input.repo`: Repository data (a result of https://docs.github.com/en/rest/reference/repos#get-a-repository)
    - `input.branches`: A list of branch (a result of https://docs.github.com/en/rest/reference/branches#list-branches)
    - `input.collaborators`: A list of collaborator (a result of https://docs.github.com/en/rest/reference/collaborators#list-repository-collaborators)
    - `input.hooks`: A list of webhooks (a result of https://docs.github.com/en/rest/reference/webhooks#list-repository-webhooks)
    - `input.teams`: A list of team (a result of https://docs.github.com/en/rest/reference/repos#list-repository-teams)
    - `input.timestamp`: Unix timestamp of scan
- Result: Put detected violation
    - `category`: Title to indicate violation category
    - `message`: Describe violation detail

#### Policy example

Example 1. Check if collaborator does not have overly permissions

```rego
package github.repo

fail[res] {
    user := input.collaborators[_]
    true == [
        user.permissions.maintain,
        user.permissions.admin,
    ][_]

    res = {
        "category": "Collaborator must not have permissions of maintain and admin",
        "message": sprintf("%s has maintain:%v admin:%v", [user.login, user.permissions.maintain, user.permissions.admin]),
    }
}
```

Example 2. Check if default branch is protected

```rego
package github.repo

fail[msg] {
	branch := input.branches[_]
    branch.name == input.repo.default_branch
    branch.protected == false
    msg := {
        "category": "default branch must be protected",
        "message": sprintf("default branch is %s", [branch.name]),
    }
}
```

### 3) [Optional] Retrieve webhook URL of Slack

`ghaudit` can notify a detected violation via Slack by incoming webhook. Setup incoming webhook according to https://api.slack.com/messaging/webhooks if you want.

## Run ghaudit

```bash
$ export GHAUDIT_APP_ID=000000
$ export GHAUDIT_INSTALL_ID=0000000
$ export GHAUDIT_PRIVATE_KEY_FILE=xxxxxx.2022-02-18.private-key.pem
$ export GHAUDIT_SLACK_WEBHOOK=https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
$ ghaudit -o [your_org_name] -p ./policy
```

### Test and debug policy

- `--dump`: Exports retrieved repository data to directory
- `--load`: Imports local repository data exported by `--dump` option
- `--log-level dump`: Output `print` result in Rego if you use local policy

Example:
```bash
(skip export environment variables)
$ ghaudit -o [your_org_name] -p ./policy --dump ./repo_data
# output repository data to ./repo_data
$ ls ./repo_data
foo-repo.json    baa-repo.json
# if something wrong, update local Rego file(s), then
$ ghaudit -o [your_org_name] -p ./policy --load ./repo_data --log-level debug
# Re-evaluate updated policy with local data rapidly and output `print` function result also
```

### Options

#### Required

- `--app-id` (`GHAUDIT_APP_ID`): GitHub App ID
- `--install-id` (`GHAUDIT_INSTALL_ID`): GitHub App install ID
- GitHub App private key: Choose either one of following:
    - `--private-key-file` (`GHAUDIT_PRIVATE_KEY_FILE`): Key file path
    - `--private-key-data` (`GHAUDIT_PRIVATE_KEY_DATA`): Key data
- Audit policy: Choose either one of following:
    - Use local Rego file(s)
        - `--policy`, `-p`: Rego policy directory. Scan `.rego` file recursively
        - `--package`: Package name of policy. Default is `github.repo`
    - Use OPA server
        - `--server`, `-s`: OPA server URL
        - `--header`, `-H`: HTTP header of inquiry request to OPA server
- `--dump`: Specify directory to dump retrieved data from GitHub
- `--load`: Specify directory to load retrieved data from GitHub

#### Optional

- `--format`, `-f`: Choose `text` or `json`.
- `--output`, `-o`: Output file. `-` means stdout.
- `--slack-webhook` (`GHAUDIT_SLACK_WEBHOOK`): Slack incoming webhook URL.
- `--fail`: Exit with non-zero when detecting violation
- `--thread`: Specify number of thread to retrieve repository meta data
- `--limit`: Specify limit number of auditing repository

## License

Apache License 2.0
