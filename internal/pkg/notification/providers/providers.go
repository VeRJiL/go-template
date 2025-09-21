package providers

import (
	"fmt"

	"github.com/VeRJiL/go-template/internal/pkg/notification/providers/email"
	"github.com/VeRJiL/go-template/internal/pkg/notification/providers/sms"
)

// Initialize provider factory functions
func init() {
	// Email provider factories
	NewSMTPEmailProvider = func(config interface{}) (EmailProvider, error) {
		return email.NewSMTPEmailProvider(config)
	}

	NewSendGridEmailProvider = func(config interface{}) (EmailProvider, error) {
		return email.NewSendGridEmailProvider(config)
	}

	NewMailgunEmailProvider = func(config interface{}) (EmailProvider, error) {
		// This would be implemented similarly to SendGrid
		return nil, fmt.Errorf("Mailgun provider not yet implemented")
	}

	NewAWSSESEmailProvider = func(config interface{}) (EmailProvider, error) {
		// This would be implemented similarly to SendGrid
		return nil, fmt.Errorf("AWS SES provider not yet implemented")
	}

	// SMS provider factories
	NewTwilioProvider = func(config interface{}) (SMSProvider, error) {
		return sms.NewTwilioSMSProvider(config)
	}

	NewAWSSNSSMSProvider = func(config interface{}) (SMSProvider, error) {
		return nil, fmt.Errorf("AWS SNS SMS provider not yet implemented")
	}

	NewNexmoProvider = func(config interface{}) (SMSProvider, error) {
		return nil, fmt.Errorf("Nexmo SMS provider not yet implemented")
	}

	NewTextMagicProvider = func(config interface{}) (SMSProvider, error) {
		return nil, fmt.Errorf("TextMagic SMS provider not yet implemented")
	}
}