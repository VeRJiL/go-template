package notification

import (
	"fmt"

	"github.com/VeRJiL/go-template/internal/pkg/notification/drivers"
)

// initializeEmailDrivers initializes email notification drivers
func (m *Manager) initializeEmailDrivers() error {
	switch m.config.Email.Provider {
	case "smtp":
		if m.config.Email.SMTP == nil {
			return fmt.Errorf("SMTP configuration is required")
		}
		driver, err := drivers.NewEmailDriver("smtp", m.config.Email.SMTP)
		if err != nil {
			return err
		}
		m.drivers["email"] = driver

	case "sendgrid":
		if m.config.Email.SendGrid == nil {
			return fmt.Errorf("SendGrid configuration is required")
		}
		driver, err := drivers.NewEmailDriver("sendgrid", m.config.Email.SendGrid)
		if err != nil {
			return err
		}
		m.drivers["email"] = driver

	case "mailgun":
		if m.config.Email.Mailgun == nil {
			return fmt.Errorf("Mailgun configuration is required")
		}
		driver, err := drivers.NewEmailDriver("mailgun", m.config.Email.Mailgun)
		if err != nil {
			return err
		}
		m.drivers["email"] = driver

	case "aws_ses":
		if m.config.Email.AWSSES == nil {
			return fmt.Errorf("AWS SES configuration is required")
		}
		driver, err := drivers.NewEmailDriver("aws_ses", m.config.Email.AWSSES)
		if err != nil {
			return err
		}
		m.drivers["email"] = driver

	default:
		return fmt.Errorf("unsupported email provider: %s", m.config.Email.Provider)
	}

	return nil
}

// initializeSMSDrivers initializes SMS notification drivers
func (m *Manager) initializeSMSDrivers() error {
	switch m.config.SMS.Provider {
	case "twilio":
		if m.config.SMS.Twilio == nil {
			return fmt.Errorf("Twilio configuration is required")
		}
		driver, err := drivers.NewSMSDriver("twilio", m.config.SMS.Twilio)
		if err != nil {
			return err
		}
		m.drivers["sms"] = driver

	case "aws_sns":
		if m.config.SMS.AWSSNS == nil {
			return fmt.Errorf("AWS SNS configuration is required")
		}
		driver, err := drivers.NewSMSDriver("aws_sns", m.config.SMS.AWSSNS)
		if err != nil {
			return err
		}
		m.drivers["sms"] = driver

	case "nexmo":
		if m.config.SMS.Nexmo == nil {
			return fmt.Errorf("Nexmo configuration is required")
		}
		driver, err := drivers.NewSMSDriver("nexmo", m.config.SMS.Nexmo)
		if err != nil {
			return err
		}
		m.drivers["sms"] = driver

	case "textmagic":
		if m.config.SMS.TextMagic == nil {
			return fmt.Errorf("TextMagic configuration is required")
		}
		driver, err := drivers.NewSMSDriver("textmagic", m.config.SMS.TextMagic)
		if err != nil {
			return err
		}
		m.drivers["sms"] = driver

	default:
		return fmt.Errorf("unsupported SMS provider: %s", m.config.SMS.Provider)
	}

	return nil
}

// initializePushDrivers initializes push notification drivers
func (m *Manager) initializePushDrivers() error {
	switch m.config.Push.Provider {
	case "fcm":
		if m.config.Push.FCM == nil {
			return fmt.Errorf("FCM configuration is required")
		}
		driver, err := drivers.NewPushDriver("fcm", m.config.Push.FCM)
		if err != nil {
			return err
		}
		m.drivers["push"] = driver

	case "apns":
		if m.config.Push.APNS == nil {
			return fmt.Errorf("APNS configuration is required")
		}
		driver, err := drivers.NewPushDriver("apns", m.config.Push.APNS)
		if err != nil {
			return err
		}
		m.drivers["push"] = driver

	case "pusher":
		if m.config.Push.Pusher == nil {
			return fmt.Errorf("Pusher configuration is required")
		}
		driver, err := drivers.NewPushDriver("pusher", m.config.Push.Pusher)
		if err != nil {
			return err
		}
		m.drivers["push"] = driver

	case "onesignal":
		if m.config.Push.OneSignal == nil {
			return fmt.Errorf("OneSignal configuration is required")
		}
		driver, err := drivers.NewPushDriver("onesignal", m.config.Push.OneSignal)
		if err != nil {
			return err
		}
		m.drivers["push"] = driver

	default:
		return fmt.Errorf("unsupported push provider: %s", m.config.Push.Provider)
	}

	return nil
}

// initializeSocialDrivers initializes social media notification drivers
func (m *Manager) initializeSocialDrivers() error {
	// WhatsApp
	if m.config.Social.WhatsApp.Enabled {
		switch m.config.Social.WhatsApp.Provider {
		case "twilio":
			if m.config.Social.WhatsApp.Twilio == nil {
				return fmt.Errorf("WhatsApp Twilio configuration is required")
			}
			driver, err := drivers.NewSocialDriver("whatsapp_twilio", m.config.Social.WhatsApp.Twilio)
			if err != nil {
				return err
			}
			m.drivers["whatsapp"] = driver

		case "whatsapp_business":
			if m.config.Social.WhatsApp.BusinessAPI == nil {
				return fmt.Errorf("WhatsApp Business API configuration is required")
			}
			driver, err := drivers.NewSocialDriver("whatsapp_business", m.config.Social.WhatsApp.BusinessAPI)
			if err != nil {
				return err
			}
			m.drivers["whatsapp"] = driver

		default:
			return fmt.Errorf("unsupported WhatsApp provider: %s", m.config.Social.WhatsApp.Provider)
		}
	}

	// Telegram
	if m.config.Social.Telegram.Enabled {
		if m.config.Social.Telegram.BotToken == "" {
			return fmt.Errorf("Telegram bot token is required")
		}
		driver, err := drivers.NewSocialDriver("telegram", m.config.Social.Telegram)
		if err != nil {
			return err
		}
		m.drivers["telegram"] = driver
	}

	// Slack
	if m.config.Social.Slack.Enabled {
		switch m.config.Social.Slack.Provider {
		case "webhook":
			if m.config.Social.Slack.WebhookURL == "" {
				return fmt.Errorf("Slack webhook URL is required")
			}
			driver, err := drivers.NewSocialDriver("slack_webhook", m.config.Social.Slack)
			if err != nil {
				return err
			}
			m.drivers["slack"] = driver

		case "bot":
			if m.config.Social.Slack.BotToken == "" {
				return fmt.Errorf("Slack bot token is required")
			}
			driver, err := drivers.NewSocialDriver("slack_bot", m.config.Social.Slack)
			if err != nil {
				return err
			}
			m.drivers["slack"] = driver

		default:
			return fmt.Errorf("unsupported Slack provider: %s", m.config.Social.Slack.Provider)
		}
	}

	// Discord
	if m.config.Social.Discord.Enabled {
		switch m.config.Social.Discord.Provider {
		case "webhook":
			if m.config.Social.Discord.WebhookURL == "" {
				return fmt.Errorf("Discord webhook URL is required")
			}
			driver, err := drivers.NewSocialDriver("discord_webhook", m.config.Social.Discord)
			if err != nil {
				return err
			}
			m.drivers["discord"] = driver

		case "bot":
			if m.config.Social.Discord.BotToken == "" {
				return fmt.Errorf("Discord bot token is required")
			}
			driver, err := drivers.NewSocialDriver("discord_bot", m.config.Social.Discord)
			if err != nil {
				return err
			}
			m.drivers["discord"] = driver

		default:
			return fmt.Errorf("unsupported Discord provider: %s", m.config.Social.Discord.Provider)
		}
	}

	return nil
}