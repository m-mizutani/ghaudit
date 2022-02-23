package infra

import (
	"github.com/m-mizutani/ghaudit/pkg/infra/githubapp"
	"github.com/m-mizutani/ghaudit/pkg/infra/notify"
	"github.com/m-mizutani/ghaudit/pkg/infra/policy"
)

type Clients struct {
	ghapp  githubapp.Client
	policy policy.Client
	slack  notify.SlackClient
}

func New(options ...Option) *Clients {
	clients := &Clients{}
	for _, opt := range options {
		opt(clients)
	}
	return clients
}

func (x *Clients) GitHubApp() githubapp.Client { return x.ghapp }
func (x *Clients) Policy() policy.Client       { return x.policy }
func (x *Clients) Slack() notify.SlackClient   { return x.slack }

type Option func(c *Clients)

func WithGitHubApp(client githubapp.Client) Option {
	return func(c *Clients) {
		c.ghapp = client
	}
}

func WithPolicy(client policy.Client) Option {
	return func(c *Clients) {
		c.policy = client
	}
}

func WithSlack(client notify.SlackClient) Option {
	return func(c *Clients) {
		c.slack = client
	}
}
