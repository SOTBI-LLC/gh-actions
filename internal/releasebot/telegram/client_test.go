package telegram

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestGetUpdatesTimeoutDoesNotLeakToken(t *testing.T) {
	t.Parallel()

	const token = "123:secret-token"

	err := telegramRequestError("getUpdates", &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/bot" + token + "/getUpdates",
		Err: context.DeadlineExceeded,
	})

	errorMessage := err.Error()
	if strings.Contains(errorMessage, token) {
		t.Fatalf("error message leaks telegram token: %s", errorMessage)
	}

	if !strings.Contains(errorMessage, "telegram getUpdates request failed") {
		t.Fatalf("unexpected error message: %s", errorMessage)
	}
}
