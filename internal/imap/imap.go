package imap

import (
	"context"
	"time"

	"github.com/emersion/go-imap"
)

type Client interface {
	Connect(server string) error
	Login(user, password string) error
	SelectMailbox(name string) error
	ListUnseenUIDs(since time.Duration) ([]uint32, error)
	FetchMessage(uid uint32) (*imap.Message, error)
	MarkSeen(uid uint32) error
	// SetConnectionTimeout overrides the connection's read timeout and
	// returns the previous value, so a caller can widen it around a
	// long-running step and restore it afterwards.
	SetConnectionTimeout(d time.Duration) time.Duration
	Close() error
	// WaitForNewMail blocks until the server signals new mail in the selected
	// mailbox (via IMAP IDLE) or ctx is cancelled.
	WaitForNewMail(ctx context.Context) error
}
