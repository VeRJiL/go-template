package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/notification/providers"
)

// TwilioProvider implements SMSProvider for Twilio
type TwilioProvider struct {
	config     *TwilioConfig
	httpClient *http.Client
	baseURL    string
}

// TwilioConfig holds Twilio configuration
type TwilioConfig struct {
	AccountSID    string `json:"account_sid" mapstructure:"account_sid"`
	AuthToken     string `json:"auth_token" mapstructure:"auth_token"`
	FromNumber    string `json:"from_number" mapstructure:"from_number"`
	StatusCallback string `json:"status_callback" mapstructure:"status_callback"`
	Timeout       int    `json:"timeout" mapstructure:"timeout"` // seconds
}

// TwilioResponse represents Twilio API response
type TwilioResponse struct {
	SID         string  `json:"sid"`
	Status      string  `json:"status"`
	To          string  `json:"to"`
	From        string  `json:"from"`
	Body        string  `json:"body"`
	Price       string  `json:"price,omitempty"`
	PriceUnit   string  `json:"price_unit,omitempty"`
	ErrorCode   *int    `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	DateCreated string  `json:"date_created"`
	DateSent    string  `json:"date_sent,omitempty"`
}

// NewTwilioSMSProvider creates a new Twilio SMS provider
func NewTwilioSMSProvider(config interface{}) (*TwilioProvider, error) {
	twilioConfig, ok := config.(*TwilioConfig)
	if !ok {
		return nil, fmt.Errorf("invalid Twilio configuration type")
	}

	if err := validateTwilioConfig(twilioConfig); err != nil {
		return nil, fmt.Errorf("Twilio configuration validation failed: %w", err)
	}

	// Set defaults
	if twilioConfig.Timeout == 0 {
		twilioConfig.Timeout = 30
	}

	httpClient := &http.Client{
		Timeout: time.Duration(twilioConfig.Timeout) * time.Second,
	}

	provider := &TwilioProvider{
		config:     twilioConfig,
		httpClient: httpClient,
		baseURL:    fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s", twilioConfig.AccountSID),
	}

	return provider, nil
}

// Send sends an SMS message via Twilio
func (p *TwilioProvider) Send(ctx context.Context, message *providers.SMSMessage) error {
	// Send to each recipient individually (Twilio doesn't support bulk SMS in a single API call)
	var lastErr error
	for _, to := range message.To {
		if err := p.sendSingle(ctx, to, message); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// SendBatch sends multiple SMS messages
func (p *TwilioProvider) SendBatch(ctx context.Context, messages []*providers.SMSMessage) error {
	var lastErr error
	for i, message := range messages {
		if err := p.Send(ctx, message); err != nil {
			lastErr = fmt.Errorf("failed to send SMS %d: %w", i, err)
		}
	}
	return lastErr
}

// GetProviderName returns the provider name
func (p *TwilioProvider) GetProviderName() string {
	return "twilio"
}

// Ping checks if Twilio API is reachable
func (p *TwilioProvider) Ping(ctx context.Context) error {
	// Try to get account info as a health check
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+".json", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Twilio API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Twilio API returned status: %d", resp.StatusCode)
	}

	return nil
}

// Close closes the Twilio provider
func (p *TwilioProvider) Close() error {
	// No cleanup needed for HTTP client
	return nil
}

// Helper methods

// sendSingle sends an SMS to a single recipient
func (p *TwilioProvider) sendSingle(ctx context.Context, to string, message *providers.SMSMessage) error {
	// Prepare form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", p.getFromNumber(message))
	data.Set("Body", message.Body)

	// Add media URLs for MMS
	if len(message.MediaURLs) > 0 {
		for i, mediaURL := range message.MediaURLs {
			data.Set(fmt.Sprintf("MediaUrl%d", i), mediaURL)
		}
	}

	// Add status callback
	statusCallback := message.StatusCallback
	if statusCallback == "" {
		statusCallback = p.config.StatusCallback
	}
	if statusCallback != "" {
		data.Set("StatusCallback", statusCallback)
	}

	// Add validity period
	if message.ValidityPeriod > 0 {
		data.Set("ValidityPeriod", fmt.Sprintf("%d", message.ValidityPeriod))
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/Messages.json",
		strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Twilio API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var twilioResp TwilioResponse
	if err := json.Unmarshal(respBody, &twilioResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		errorMsg := twilioResp.ErrorMessage
		if errorMsg == "" {
			errorMsg = string(respBody)
		}
		return fmt.Errorf("Twilio API error (status %d): %s", resp.StatusCode, errorMsg)
	}

	if twilioResp.ErrorCode != nil && *twilioResp.ErrorCode != 0 {
		return fmt.Errorf("Twilio error %d: %s", *twilioResp.ErrorCode, twilioResp.ErrorMessage)
	}

	return nil
}

// getFromNumber returns the from number
func (p *TwilioProvider) getFromNumber(message *providers.SMSMessage) string {
	if message.From != "" {
		return message.From
	}
	return p.config.FromNumber
}

// validateTwilioConfig validates Twilio configuration
func validateTwilioConfig(config *TwilioConfig) error {
	if config.AccountSID == "" {
		return fmt.Errorf("Twilio Account SID is required")
	}

	if config.AuthToken == "" {
		return fmt.Errorf("Twilio Auth Token is required")
	}

	if config.FromNumber == "" {
		return fmt.Errorf("Twilio from number is required")
	}

	return nil
}