package auth

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Sender delivers a one-time code to a phone number (E.164).
type Sender interface {
	SendCode(ctx context.Context, phone, code string) error
}

// LogSender prints codes to the log. Development only: it must never be
// selected when APP_ENV is not "development".
type LogSender struct{}

func (LogSender) SendCode(_ context.Context, phone, code string) error {
	log.Printf("[dev] OTP for %s is %s", phone, code)
	return nil
}

// Kavenegar sends codes through the verify/lookup template API.
type Kavenegar struct {
	APIKey   string
	Template string
	BaseURL  string // defaults to https://api.kavenegar.com
	Client   *http.Client
}

func (k Kavenegar) SendCode(ctx context.Context, phone, code string) error {
	base := k.BaseURL
	if base == "" {
		base = "https://api.kavenegar.com"
	}
	client := k.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	q := url.Values{}
	q.Set("receptor", LocalFormat(phone))
	q.Set("token", code)
	q.Set("template", k.Template)
	endpoint := fmt.Sprintf("%s/v1/%s/verify/lookup.json?%s", base, url.PathEscape(k.APIKey), q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("kavenegar request failed")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		// Never include the URL: it contains the API key and the code.
		return fmt.Errorf("kavenegar returned status %d", resp.StatusCode)
	}
	return nil
}
