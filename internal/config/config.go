package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultHTTPAddr        = ":8080"
	defaultTelegramAPIBase = "https://api.telegram.org"
	defaultGitHubAPIBase   = "https://api.github.com"
	DefaultWorkflowFile    = "restart.yaml"
)

// Params contains release bot runtime settings.
type Params struct {
	HTTPAddr                   string
	TelegramBotToken           string
	TelegramChatID             string
	TelegramAllowedUserIDs     map[int64]struct{}
	TelegramWebhookSecretToken string
	TelegramAPIBaseURL         string
	GitHubToken                string
	GitHubAPIBaseURL           string
	BuildNotificationSecret    string
	WorkflowFile               string
	EnableLongPolling          bool
}

func parseAllowedUserIDs(raw string) (map[int64]struct{}, error) {
	allowedUserIDs := make(map[int64]struct{})

	for part := range strings.SplitSeq(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse TELEGRAM_ALLOWED_USER_IDS: %w", err)
		}

		allowedUserIDs[userID] = struct{}{}
	}

	return allowedUserIDs, nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}

func LoadConfig() (*Params, error) {
	allowedUserIDs, err := parseAllowedUserIDs(os.Getenv("TELEGRAM_ALLOWED_USER_IDS"))
	if err != nil {
		return nil, err
	}

	config := Params{
		HTTPAddr:                   envOrDefault("RELEASEBOT_HTTP_ADDR", defaultHTTPAddr),
		TelegramBotToken:           os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:             os.Getenv("TELEGRAM_CHAT_ID"),
		TelegramAllowedUserIDs:     allowedUserIDs,
		TelegramWebhookSecretToken: os.Getenv("TELEGRAM_WEBHOOK_SECRET_TOKEN"),
		TelegramAPIBaseURL:         envOrDefault("TELEGRAM_API_BASE_URL", defaultTelegramAPIBase),
		GitHubToken:                os.Getenv("GITHUB_TOKEN"),
		GitHubAPIBaseURL:           envOrDefault("GITHUB_API_BASE_URL", defaultGitHubAPIBase),
		BuildNotificationSecret:    os.Getenv("RELEASEBOT_SHARED_SECRET"),
		WorkflowFile:               envOrDefault("RELEASEBOT_WORKFLOW_FILE", DefaultWorkflowFile),
		EnableLongPolling:          strings.EqualFold(os.Getenv("RELEASEBOT_LONG_POLLING"), "true"),
	}

	if config.TelegramBotToken == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	if config.TelegramChatID == "" {
		return nil, errors.New("TELEGRAM_CHAT_ID is required")
	}

	if len(config.TelegramAllowedUserIDs) == 0 {
		return nil, errors.New("TELEGRAM_ALLOWED_USER_IDS is required")
	}

	if config.GitHubToken == "" {
		return nil, errors.New("GITHUB_TOKEN is required")
	}

	if config.BuildNotificationSecret == "" {
		return nil, errors.New("RELEASEBOT_SHARED_SECRET is required")
	}

	return &config, nil
}
