package config

import (
	"strings"
	"testing"
)

func TestParseAllowedUserIDs(t *testing.T) {
	t.Parallel()

	allowedUserIDs, err := parseAllowedUserIDs("123, 456")
	if err != nil {
		t.Fatalf("parse allowed user ids: %v", err)
	}

	for _, userID := range []int64{123, 456} {
		if _, ok := allowedUserIDs[userID]; !ok {
			t.Fatalf("expected user id %d to be allowed", userID)
		}
	}
}

func TestParseAllowedUserIDsRejectsInvalidID(t *testing.T) {
	t.Parallel()

	if _, err := parseAllowedUserIDs("123, not-a-user-id"); err == nil {
		t.Fatalf("expected invalid user id error")
	}
}

func TestLoadConfigRequiresAccessControlSettings(t *testing.T) {
	baseEnv := map[string]string{
		"TELEGRAM_BOT_TOKEN":        "telegram-token",
		"TELEGRAM_CHAT_ID":          "100",
		"TELEGRAM_ALLOWED_USER_IDS": "42",
		"GITHUB_TOKEN":              "github-token",
		"RELEASEBOT_SHARED_SECRET":  "build-secret",
	}

	tests := []struct {
		name       string
		missingEnv string
		wantErr    string
	}{
		{
			name:       "requires telegram allowed user ids",
			missingEnv: "TELEGRAM_ALLOWED_USER_IDS",
			wantErr:    "TELEGRAM_ALLOWED_USER_IDS is required",
		},
		{
			name:       "requires releasebot shared secret",
			missingEnv: "RELEASEBOT_SHARED_SECRET",
			wantErr:    "RELEASEBOT_SHARED_SECRET is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, value := range baseEnv {
				t.Setenv(name, value)
			}

			t.Setenv(tt.missingEnv, "")

			if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected %q error, got %v", tt.wantErr, err)
			}
		})
	}
}
