package server

import (
	"errors"

	"github.com/gregdel/pushover"
)

type notifyClient struct {
	app       *pushover.Pushover
	recipient *pushover.Recipient
}

func newNotifyClient(appToken, recipientToken string) (*notifyClient, error) {
	if appToken == "" {
		return nil, errors.New("missing required app_token")
	}
	if recipientToken == "" {
		return nil, errors.New("missing required recipient_token")
	}

	return &notifyClient{
		app:       pushover.New(appToken),
		recipient: pushover.NewRecipient(recipientToken),
	}, nil
}

func (c *notifyClient) send(title, message string) error {
	msg := pushover.NewMessageWithTitle(message, title)
	_, err := c.app.SendMessage(msg, c.recipient)
	return err
}
