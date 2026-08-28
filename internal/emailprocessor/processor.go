package emailprocessor

import (
	"netflix-household-validator/internal/mailparse"
	"time"

	imapclient "netflix-household-validator/internal/imap"
	"netflix-household-validator/internal/logging"
	"netflix-household-validator/internal/models"

	"netflix-household-validator/internal/netflix"
)

// EmailValidityWindow Netflix typically sends verification emails immediately,so a 15-minute window is reasonable to
// account for delays without processing stale emails.
const EmailValidityWindow = 15 * time.Minute

// browserAutomationTimeout is the IMAP connection read timeout used while
// the browser automation step runs. It must comfortably exceed the worst
// observed browser-flow duration (up to ~90s in production logs).
const browserAutomationTimeout = 3 * time.Minute

// ProcessingStats tracks email processing statistics for a cycle
type ProcessingStats struct {
	Total          int
	Processed      int
	Ignored        int
	Failed         int
	TooOld         int
	NoMatchingLink int
}

type Processor struct {
	imapClient     imapclient.Client
	netflixService *netflix.Service
}

// NewProcessor creates a new Processor instance with the provided IMAP client and Netflix service
func NewProcessor(imapClient imapclient.Client, netflixService *netflix.Service) *Processor {
	return &Processor{
		imapClient:     imapClient,
		netflixService: netflixService,
	}
}

// ProcessEmail orchestrates the complete email processing workflow:
// fetch → parse → validate age → handle → mark as seen
// Returns a boolean indicating if the email was successfully handled and should update stats
func (p *Processor) ProcessEmail(uid uint32) (handled bool, ignored bool, err error) {
	// Fetch message from IMAP
	msg, err := p.imapClient.FetchMessage(uid)
	if err != nil {
		return false, false, err
	}

	// Parse email to normalized structure
	email, err := mailparse.Parse(msg)
	if err != nil {
		logging.Log.WithField("trace_id", "unknown").Errorf("Error parsing email UID %d: %v", uid, err)
		return false, false, err
	}

	locallog := logging.Log.WithField("trace_id", email.TraceID)

	// Validate email age (15 minutes window)
	if !p.isEmailValid(email) {
		locallog.Infof("Message UID %d is older than %v (date: %v), skipping", uid, EmailValidityWindow, email.InternalDate)
		return false, false, nil
	}

	// Handle email with Netflix service (filters, browser automation).
	// This can legitimately take well over 30s (cold browser launch +
	// Netflix's own page render time), during which the IMAP connection
	// sits idle. Widen the connection's read timeout for the duration so
	// go-imap's background reader doesn't tear down a perfectly healthy
	// connection out from under us, then restore the normal timeout.
	prevTimeout := p.imapClient.SetConnectionTimeout(browserAutomationTimeout)
	handled = p.netflixService.HandleEmail(email)
	p.imapClient.SetConnectionTimeout(prevTimeout)

	// Mark as seen only if successfully handled
	if handled {
		if err := p.imapClient.MarkSeen(uid); err != nil {
			locallog.Errorf("Error marking message UID %d as seen: %v", uid, err)
		}
		return true, false, nil
	}

	return false, true, nil
}

// isEmailValid checks if email is within the validity window (15 minutes)
func (p *Processor) isEmailValid(email *models.Email) bool {
	return p.isEmailValidAt(email, time.Now())
}

// isEmailValidAt allows testing with a fixed "now" time for deterministic unit tests
func (p *Processor) isEmailValidAt(email *models.Email, now time.Time) bool {
	if email.InternalDate.IsZero() {
		return true
	}

	cutoff := now.Add(-EmailValidityWindow)
	return !email.InternalDate.Before(cutoff) // inclusif
}
