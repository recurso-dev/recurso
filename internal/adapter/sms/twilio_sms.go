package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/recur-so/recurso/internal/core/port"
)

// TwilioSMSSender sends SMS via Twilio REST API
type TwilioSMSSender struct {
	accountSID string
	authToken  string
	fromNumber string
	httpClient *http.Client
}

func NewTwilioSMSSender(accountSID, authToken, fromNumber string) port.SMSSender {
	return &TwilioSMSSender{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{},
	}
}

func (s *TwilioSMSSender) Send(ctx context.Context, msg port.SMSMessage) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.accountSID)

	data := url.Values{}
	data.Set("To", msg.To)
	data.Set("From", s.fromNumber)
	data.Set("Body", msg.Body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create twilio request: %w", err)
	}

	req.SetBasicAuth(s.accountSID, s.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send twilio SMS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}

	return nil
}
