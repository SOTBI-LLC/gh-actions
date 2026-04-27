package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SOTBI-LLC/gh-actions/internal/config"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/server"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/telegram"
)

func TestBuildNotificationAndWebhookDispatch(t *testing.T) {
	t.Parallel()

	var (
		telegramMu    sync.Mutex
		telegramCalls []telegramCall
		dispatched    struct {
			Ref    string            `json:"ref"`
			Inputs map[string]string `json:"inputs"`
		}
	)

	telegramAPI := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := telegramMethodFromPath(r.URL.Path)
			switch method {
			case "sendMessage", "editMessageReplyMarkup", "answerCallbackQuery":
			default:
				t.Fatalf("unexpected telegram path: %s", r.URL.Path)
			}

			var payload map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode telegram %s: %v", method, err)
			}

			telegramMu.Lock()

			telegramCalls = append(telegramCalls, telegramCall{
				Method:  method,
				Payload: payload,
			})
			telegramMu.Unlock()
			writeTelegramResponse(w, true)
		}),
	)
	defer telegramAPI.Close()

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/SOTBI-LLC/service/actions/workflows/restart.yaml/dispatches" {
			t.Fatalf("unexpected github path: %s", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer github-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		if err := json.NewDecoder(r.Body).Decode(&dispatched); err != nil {
			t.Fatalf("decode github dispatch: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer githubAPI.Close()

	mux := newTestMux(telegramAPI.URL, githubAPI.URL)
	notification := domain.BuildNotification{
		Repository:    "SOTBI-LLC/service",
		Ref:           "main",
		Branch:        "main",
		SHA:           "abc",
		Tag:           "v1.2.3",
		Actor:         "octocat",
		CommitMessage: "ship release",
		RunURL:        "https://github.com/SOTBI-LLC/service/actions/runs/1",
	}

	response := performJSONRequest(
		t,
		mux,
		http.MethodPost,
		"/build-notifications",
		notification,
		map[string]string{
			"X-Releasebot-Secret": "build-secret",
		},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"unexpected build notification status: %d body=%s",
			response.Code,
			response.Body.String(),
		)
	}

	calls := telegramCallsSnapshot(&telegramMu, telegramCalls)
	if len(calls) != 1 || calls[0].Method != "sendMessage" {
		t.Fatalf("unexpected telegram calls after build notification: %+v", calls)
	}

	messageText := decodeTelegramPayloadValue[string](t, calls[0], "text")
	if !strings.Contains(messageText, "ship release") {
		t.Fatalf("telegram message does not include commit message: %s", messageText)
	}

	releaseMarkup := decodeTelegramPayloadValue[telegram.InlineKeyboardMarkup](
		t,
		calls[0],
		"reply_markup",
	)
	releaseID := extractReleaseID(t, releaseMarkup)
	releaseUpdate := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "release-callback",
			From: telegram.User{ID: 42},
			Message: &telegram.Message{
				MessageID: 10,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 100},
			},
			Data: domain.ActionRelease + ":" + releaseID,
		},
	}

	response = performJSONRequest(
		t,
		mux,
		http.MethodPost,
		"/telegram/webhook",
		releaseUpdate,
		map[string]string{
			"X-Telegram-Bot-Api-Secret-Token": "webhook-secret",
		},
	)
	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"unexpected release webhook status: %d body=%s",
			response.Code,
			response.Body.String(),
		)
	}

	calls = telegramCallsSnapshot(&telegramMu, telegramCalls)
	if len(calls) != 3 || calls[1].Method != "editMessageReplyMarkup" ||
		calls[2].Method != "answerCallbackQuery" {
		t.Fatalf("unexpected telegram calls after release callback: %+v", calls)
	}

	environmentMarkup := decodeTelegramPayloadValue[telegram.InlineKeyboardMarkup](
		t,
		calls[1],
		"reply_markup",
	)
	assertEnvironmentKeyboard(t, environmentMarkup, releaseID)

	callbackText := decodeTelegramPayloadValue[string](t, calls[2], "text")
	if callbackText != "Choose environment" {
		t.Fatalf("unexpected release callback answer: %s", callbackText)
	}

	deployUpdate := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "deploy-callback",
			From: telegram.User{ID: 42},
			Message: &telegram.Message{
				MessageID: 10,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 100},
			},
			Data: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentDev,
		},
	}

	response = performJSONRequest(
		t,
		mux,
		http.MethodPost,
		"/telegram/webhook",
		deployUpdate,
		map[string]string{
			"X-Telegram-Bot-Api-Secret-Token": "webhook-secret",
		},
	)
	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"unexpected deploy webhook status: %d body=%s",
			response.Code,
			response.Body.String(),
		)
	}

	if dispatched.Ref != "main" || dispatched.Inputs["tag"] != "v1.2.3" ||
		dispatched.Inputs["environment"] != domain.EnvironmentDev {
		t.Fatalf("unexpected dispatch payload: %+v", dispatched)
	}

	calls = telegramCallsSnapshot(&telegramMu, telegramCalls)
	if len(calls) != 5 || calls[3].Method != "editMessageReplyMarkup" ||
		calls[4].Method != "answerCallbackQuery" {
		t.Fatalf("unexpected telegram calls after deploy callback: %+v", calls)
	}

	deployedMarkup := decodeTelegramPayloadValue[telegram.InlineKeyboardMarkup](
		t,
		calls[3],
		"reply_markup",
	)
	assertPostDeployEnvironmentKeyboard(
		t,
		deployedMarkup,
		releaseID,
		true,
		false,
	)

	callbackText = decodeTelegramPayloadValue[string](t, calls[4], "text")
	if callbackText != "Deploy started for "+domain.EnvironmentDev {
		t.Fatalf("unexpected deploy callback answer: %s", callbackText)
	}

	deployProdUpdate := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "deploy-prod-callback",
			From: telegram.User{ID: 42},
			Message: &telegram.Message{
				MessageID: 10,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 100},
			},
			Data: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentProd,
		},
	}

	response = performJSONRequest(
		t,
		mux,
		http.MethodPost,
		"/telegram/webhook",
		deployProdUpdate,
		map[string]string{
			"X-Telegram-Bot-Api-Secret-Token": "webhook-secret",
		},
	)
	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"unexpected second deploy webhook status: %d body=%s",
			response.Code,
			response.Body.String(),
		)
	}

	if dispatched.Inputs["environment"] != domain.EnvironmentProd {
		t.Fatalf("unexpected second dispatch: %+v", dispatched)
	}

	calls = telegramCallsSnapshot(&telegramMu, telegramCalls)
	if len(calls) != 7 || calls[5].Method != "editMessageReplyMarkup" ||
		calls[6].Method != "answerCallbackQuery" {
		t.Fatalf("unexpected telegram calls after second deploy: %+v", calls)
	}

	bothDoneMarkup := decodeTelegramPayloadValue[telegram.InlineKeyboardMarkup](
		t,
		calls[5],
		"reply_markup",
	)
	assertPostDeployEnvironmentKeyboard(
		t,
		bothDoneMarkup,
		releaseID,
		true,
		true,
	)
}

func TestWebhookRejectsUnauthorizedUser(t *testing.T) {
	t.Parallel()

	telegramAPI := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeTelegramResponse(w, true)
		}),
	)
	defer telegramAPI.Close()

	mux := newTestMux(telegramAPI.URL, "http://github.invalid")
	update := telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "release-callback",
			From: telegram.User{ID: 7},
			Data: domain.ActionRelease + ":abc123",
		},
	}

	response := performJSONRequest(
		t,
		mux,
		http.MethodPost,
		"/telegram/webhook",
		update,
		map[string]string{
			"X-Telegram-Bot-Api-Secret-Token": "webhook-secret",
		},
	)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body.String())
	}
}

func TestBuildNotificationRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	mux := newTestMux("http://telegram.invalid", "http://github.invalid")
	notification := domain.BuildNotification{
		Repository:    "SOTBI-LLC/service",
		Ref:           "main",
		SHA:           "abc",
		Tag:           "v1.2.3",
		Actor:         "octocat",
		CommitMessage: "ship release",
		RunURL:        "https://github.com/SOTBI-LLC/service/actions/runs/1",
	}

	tests := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "missing secret",
		},
		{
			name: "wrong secret",
			headers: map[string]string{
				"X-Releasebot-Secret": "wrong-secret",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := performJSONRequest(
				t,
				mux,
				http.MethodPost,
				"/build-notifications",
				notification,
				tt.headers,
			)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestWebhookAllowsTelegramExtraFields(t *testing.T) {
	t.Parallel()

	mux := newTestMux("http://telegram.invalid", "http://github.invalid")
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/telegram/webhook",
		strings.NewReader(`{"update_id":1,"message":{"text":"ignored"}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "webhook-secret")

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)

	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body.String())
	}
}

func newTestMux(telegramURL, githubURL string) http.Handler {
	bot := server.New(config.Params{
		HTTPAddr:                   "127.0.0.1:0",
		TelegramBotToken:           "telegram-token",
		TelegramChatID:             "100",
		TelegramAllowedUserIDs:     map[int64]struct{}{42: {}},
		TelegramWebhookSecretToken: "webhook-secret",
		TelegramAPIBaseURL:         telegramURL,
		GitHubToken:                "github-token",
		GitHubAPIBaseURL:           githubURL,
		BuildNotificationSecret:    "build-secret",
		WorkflowFile:               config.DefaultWorkflowFile,
	}, slog.Default(), http.DefaultClient)

	mux := http.NewServeMux()
	bot.RegisterRoutes(mux)

	return mux
}

func performJSONRequest(
	t *testing.T,
	handler http.Handler,
	method, path string,
	payload any,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	for name, value := range headers {
		req.Header.Set(name, value)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	return response
}

func writeTelegramResponse(w http.ResponseWriter, result bool) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"result": result,
	})
}

func extractReleaseID(t *testing.T, markup telegram.InlineKeyboardMarkup) string {
	t.Helper()

	if len(markup.InlineKeyboard) != 1 || len(markup.InlineKeyboard[0]) != 1 {
		t.Fatalf("unexpected keyboard: %+v", markup)
	}

	callbackData, err := domain.ParseCallbackData(markup.InlineKeyboard[0][0].CallbackData)
	if err != nil {
		t.Fatalf("parse release callback: %v", err)
	}

	return callbackData.ReleaseID
}

type telegramCall struct {
	Method  string
	Payload map[string]json.RawMessage
}

func telegramMethodFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

func telegramCallsSnapshot(mu *sync.Mutex, calls []telegramCall) []telegramCall {
	mu.Lock()
	defer mu.Unlock()

	snapshot := make([]telegramCall, len(calls))
	copy(snapshot, calls)

	return snapshot
}

func decodeTelegramPayloadValue[T any](t *testing.T, call telegramCall, field string) T {
	t.Helper()

	raw, ok := call.Payload[field]
	if !ok {
		t.Fatalf("telegram %s payload missing field %q: %+v", call.Method, field, call.Payload)
	}

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode telegram %s payload field %q: %v", call.Method, field, err)
	}

	return value
}

func assertEnvironmentKeyboard(
	t *testing.T,
	markup telegram.InlineKeyboardMarkup,
	releaseID string,
) {
	t.Helper()

	if len(markup.InlineKeyboard) != 1 || len(markup.InlineKeyboard[0]) != 2 {
		t.Fatalf("unexpected environment keyboard: %+v", markup)
	}

	expectedButtons := []telegram.InlineKeyboardButton{
		{
			Text:         domain.EnvironmentDev,
			CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentDev,
		},
		{
			Text:         domain.EnvironmentProd,
			CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentProd,
		},
	}
	for index, expectedButton := range expectedButtons {
		if markup.InlineKeyboard[0][index] != expectedButton {
			t.Fatalf(
				"unexpected environment button at %d: %+v",
				index,
				markup.InlineKeyboard[0][index],
			)
		}
	}
}

func assertPostDeployEnvironmentKeyboard(
	t *testing.T,
	markup telegram.InlineKeyboardMarkup,
	releaseID string,
	wantDevDone, wantProdDone bool,
) {
	t.Helper()

	if len(markup.InlineKeyboard) != 1 || len(markup.InlineKeyboard[0]) != 2 {
		t.Fatalf("unexpected post-deploy keyboard: %+v", markup)
	}

	dev := markup.InlineKeyboard[0][0]
	prod := markup.InlineKeyboard[0][1]

	if wantDevDone {
		if dev.Text != "✓ "+domain.EnvironmentDev || dev.CallbackData != domain.ActionNoop {
			t.Fatalf("unexpected dev button after deploy: %+v", dev)
		}
	} else {
		if dev != (telegram.InlineKeyboardButton{
			Text:         domain.EnvironmentDev,
			CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentDev,
		}) {
			t.Fatalf("unexpected dev button: %+v", dev)
		}
	}

	if wantProdDone {
		if prod.Text != "✓ "+domain.EnvironmentProd || prod.CallbackData != domain.ActionNoop {
			t.Fatalf("unexpected prod button after deploy: %+v", prod)
		}
	} else {
		if prod != (telegram.InlineKeyboardButton{
			Text:         domain.EnvironmentProd,
			CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentProd,
		}) {
			t.Fatalf("unexpected prod button: %+v", prod)
		}
	}
}
