package notify

import (
	"encoding/json"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/goerr"
	"github.com/slack-go/slack"
)

type SlackClient interface {
	Post(ctx *types.Context, msg *slack.WebhookMessage) error
}

type webhookClient struct {
	url string
}

func NewSlackWebhook(url string) *webhookClient {
	return &webhookClient{
		url: url,
	}
}

func (x *webhookClient) Post(ctx *types.Context, msg *slack.WebhookMessage) error {
	if err := slack.PostWebhookContext(ctx, x.url, msg); err != nil {
		raw, _ := json.Marshal(msg)
		return goerr.Wrap(err).With("body", string(raw))
	}
	return nil
}
