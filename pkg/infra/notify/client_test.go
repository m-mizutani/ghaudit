package notify_test

import (
	"os"
	"testing"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/infra/notify"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
)

func TestSlackClient(t *testing.T) {
	url, ok := os.LookupEnv(types.EnvSlackWebhook)
	if !ok {
		t.Skip(types.EnvSlackWebhook + " is not set")
	}
	client := notify.NewSlackWebhook(url)

	msg := &slack.WebhookMessage{
		Username: "ghaudit",
		Attachments: []slack.Attachment{
			{
				Color:      "#E01E5A",
				AuthorName: "ghaudit",
				AuthorLink: "https://github.com/m-mizutani/ghaudit",

				Blocks: slack.Blocks{
					BlockSet: []slack.Block{
						slack.NewHeaderBlock(
							slack.NewTextBlockObject(slack.PlainTextType, ":warning: hello", true, false),
						),
						slack.SectionBlock{
							Type: slack.MBTSection,
							Text: &slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: "block section text",
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, client.Post(types.NewContext(), msg))
}
