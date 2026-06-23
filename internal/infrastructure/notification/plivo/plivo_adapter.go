package plivo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PlivoAdapter struct {
	AuthID    string
	AuthToken string
	Source    string // The sender ID or number
}

func NewPlivoAdapter(authID, authToken, source string) *PlivoAdapter {
	return &PlivoAdapter{
		AuthID:    authID,
		AuthToken: authToken,
		Source:    source,
	}
}

func (p *PlivoAdapter) SendSMS(to, content string) error {
	url := fmt.Sprintf("https://api.plivo.com/v1/Account/%s/Message/", p.AuthID)

	payload := map[string]string{
		"src":  p.Source,
		"dst":  to,
		"text": content,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal plivo payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.AuthID, p.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to plivo: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("plivo api returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
