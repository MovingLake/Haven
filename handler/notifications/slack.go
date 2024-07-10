package notifications

import (
	"github.com/slack-go/slack"
)

type Sender interface {
	SendMessage(message string) error
	IsActive() bool
}

type SlackSender struct {
	ChannelID string
	cli       SlackCli
}

type SlackCli interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

func NewSlackSender(token, channelID string) *SlackSender {
	// Create a new Slack client
	api := slack.New(token)
	return &SlackSender{
		ChannelID: channelID,
		cli:       api,
	}
}

func (s *SlackSender) IsActive() bool {
	return s.cli != nil && s.ChannelID != ""
}

// SendMessageToSlack sends a message to a Slack channel.
// Ignore this function for coverage since it mostly uses external dependencies.
func (s *SlackSender) SendMessage(message string) error {
	// Set the message options
	options := []slack.MsgOption{
		slack.MsgOptionText(message, false),
	}

	// Send the message
	_, _, err := s.cli.PostMessage(s.ChannelID, options...)
	if err != nil {
		return err
	}

	return nil
}
