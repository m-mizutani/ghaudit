package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/ghaudit/pkg/utils"
	"github.com/slack-go/slack"
)

type auditResult struct {
	Repos       []*github.Repository
	Records     map[string][]*auditRecord
	StartedAt   time.Time
	CompletedAt time.Time
}

func newAuditResult(repos []*github.Repository, startedAt time.Time) *auditResult {
	return &auditResult{
		Repos:     repos,
		Records:   map[string][]*auditRecord{},
		StartedAt: startedAt,
	}
}

const (
	slackMessageTitle = "GitHub Audit: evaluation completed"
)

func (x *auditResult) summarySection() *slack.SectionBlock {
	diff := x.CompletedAt.Sub(x.StartedAt)
	return slack.NewSectionBlock(nil,
		[]*slack.TextBlockObject{
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("*Scanned*: %d repos", len(x.Repos)),
				false, false,
			),
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("*Elapsed*: %s", diff.String()),
				false, false,
			),
		},
		nil,
	)
}

func (x *auditResult) Add(records ...*auditRecord) {
	for _, r := range records {
		x.Records[r.Category] = append(x.Records[r.Category], r)
	}
}

func (x *auditResult) buildViolationSlackBlocks() []slack.Block {
	const listLimit = 16
	var blocks []slack.Block
	for cat, records := range x.Records {
		lines := []string{fmt.Sprintf("Policy: *%s*", cat)}
		for i := 0; i < listLimit && i < len(records); i++ {
			r := records[i]
			msg := fmt.Sprintf("- <%s|%s>", r.Repo.GetHTMLURL(), r.Repo.GetFullName())
			if r.Message != "" {
				msg += ": " + r.Message
			}

			lines = append(lines, msg)
		}
		if more := len(records) - listLimit; more > 0 {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("and more %d repos", more))
		}

		blocks = append(blocks, []slack.Block{
			slack.NewDividerBlock(),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, strings.Join(lines, "\n"), false, false), nil, nil,
			),
		}...)
	}
	return blocks
}

func (x *auditResult) createNoViolationSlackMessage() *slack.WebhookMessage {
	utils.Logger.Trace("creating NO violation slack message")

	return &slack.WebhookMessage{
		Text: slackMessageTitle,
		Attachments: []slack.Attachment{
			{
				Color: "#2EB67D",
				Blocks: slack.Blocks{
					BlockSet: []slack.Block{
						slack.NewHeaderBlock(
							slack.NewTextBlockObject(
								slack.PlainTextType,
								`:white_check_mark: GitHub Audit: No violation detected`,
								false, false),
						),
						x.summarySection(),
					},
				},
			},
		},
	}
}

func (x *auditResult) createPolicyViolationSlackMessage() *slack.WebhookMessage {
	utils.Logger.Trace("creating violation slack message")

	blocks := append([]slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(
				slack.PlainTextType,
				fmt.Sprintf(":rotating_light: %d policy violated", len(x.Records)),
				false, false),
		),
		x.summarySection(),
	}, x.buildViolationSlackBlocks()...)

	return &slack.WebhookMessage{
		Text: slackMessageTitle,
		Attachments: []slack.Attachment{
			{
				Color: "#E01E5A",
				Blocks: slack.Blocks{
					BlockSet: blocks,
				},
			},
		},
	}
}

func (x *Usecase) output(ctx *types.Context, result *auditResult) error {
	// console output
	if len(result.Records) > 0 {
		fmt.Printf("\n===== %d violation detected =====\n", len(result.Records))
		for category, records := range result.Records {
			fmt.Printf("[%s]\n", category)
			for _, record := range records {
				fmt.Printf("- %s: %s\n", record.Repo.GetFullName(), record.Message)
			}
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("\n----- No violation detected -----\n\n")
	}

	// slack notification
	if client := x.clients.Slack(); client != nil {
		utils.Logger.Trace("sending slack message")
		var msg *slack.WebhookMessage
		if len(result.Records) > 0 {
			msg = result.createPolicyViolationSlackMessage()
		} else {
			msg = result.createNoViolationSlackMessage()
		}

		if msg != nil {
			if err := client.Post(ctx, msg); err != nil {
				return err
			}
		}
	}

	if len(result.Records) > 0 {
		return types.ErrViolationDetected
	}
	return nil
}
