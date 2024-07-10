package notifications

import (
	"errors"
	"testing"

	"github.com/slack-go/slack"
)

type fakeSlackCli struct {
	err error
}

func (f *fakeSlackCli) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return "", "", f.err
}

func TestIsActive(t *testing.T) {
	s := NewSlackSender("token", "channel")
	if !s.IsActive() {
		t.Errorf("expected sender to be active")
	}
	s2 := NewSlackSender("", "")
	if s2.IsActive() {
		t.Errorf("expected sender to be inactive")
	}
}

func TestPostMessage(t *testing.T) {
	s := NewSlackSender("token", "channel")
	s.cli = &fakeSlackCli{}
	err := s.SendMessage("test message")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	s.cli = &fakeSlackCli{err: slack.ErrInvalidConfiguration}
	err = s.SendMessage("test message")
	if !errors.Is(err, slack.ErrInvalidConfiguration) {
		t.Errorf("expected error, got %v", err)
	}
}
