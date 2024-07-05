package notifications

import (
	"github.com/slack-go/slack"
)

type Sender interface {
	SendMessage(message string) error
	IsActive() bool
}

type SlackSender struct {
	Token     string
	ChannelID string
}

func NewSlackSender(token, channelID string) *SlackSender {
	return &SlackSender{
		Token:     token,
		ChannelID: channelID,
	}
}

func (s *SlackSender) IsActive() bool {
	return s.Token != "" && s.ChannelID != ""
}

// SendMessageToSlack sends a message to a Slack channel.
func (s *SlackSender) SendMessage(message string) error {
	// Create a new Slack client
	api := slack.New(s.Token)

	// Set the message options
	options := []slack.MsgOption{
		slack.MsgOptionText(message, false),
	}

	// Send the message
	_, _, err := api.PostMessage(s.ChannelID, options...)
	if err != nil {
		return err
	}

	return nil
}
