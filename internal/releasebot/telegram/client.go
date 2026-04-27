package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxTelegramResponseBodyBytes = 64 * 1024

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseURL string
	token   string
	http    Doer
}

func NewClient(baseURL, token string, httpClient Doer) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    httpClient,
	}
}

func (c *Client) SendMessage(
	ctx context.Context,
	chatID, text string,
	replyMarkup InlineKeyboardMarkup,
) error {
	payload := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "Markdown",
		"reply_markup": replyMarkup,
	}

	return c.do(ctx, "sendMessage", payload, nil)
}

func (c *Client) EditMessageReplyMarkup(
	ctx context.Context,
	chatID, messageID int64,
	replyMarkup InlineKeyboardMarkup,
) error {
	payload := map[string]any{
		"chat_id":      chatID,
		"message_id":   messageID,
		"reply_markup": replyMarkup,
	}

	return c.do(ctx, "editMessageReplyMarkup", payload, nil)
}

func (c *Client) AnswerCallback(
	ctx context.Context,
	callbackID, text string,
	showAlert bool,
) error {
	payload := map[string]any{
		"callback_query_id": callbackID,
		"text":              text,
		"show_alert":        showAlert,
	}

	return c.do(ctx, "answerCallbackQuery", payload, nil)
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	payload := map[string]any{
		"offset":  offset,
		"timeout": timeout,
		"allowed_updates": []string{
			"callback_query",
		},
	}

	var updates []Update
	if err := c.do(ctx, "getUpdates", payload, &updates); err != nil {
		return nil, err
	}

	return updates, nil
}

func (c *Client) do(ctx context.Context, method string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/bot"+c.token+"/"+method,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return telegramRequestError(method, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBodyBytes))
	if err != nil {
		return fmt.Errorf("telegram %s read response: %w", method, err)
	}

	var apiResponse APIResponse[json.RawMessage]
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return fmt.Errorf(
			"telegram %s decode response: status=%d body=%s: %w",
			method,
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
			err,
		)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices ||
		!apiResponse.OK {
		return fmt.Errorf(
			"telegram %s failed: status=%d error_code=%d description=%s",
			method,
			resp.StatusCode,
			apiResponse.ErrorCode,
			apiResponse.Description,
		)
	}

	if result == nil || len(apiResponse.Result) == 0 {
		return nil
	}

	if err := json.Unmarshal(apiResponse.Result, result); err != nil {
		return fmt.Errorf("telegram %s decode result: %w", method, err)
	}

	return nil
}

func telegramRequestError(method string, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Err != nil {
			return fmt.Errorf(
				"telegram %s request failed: op=%s error=%v",
				method,
				urlErr.Op,
				urlErr.Err,
			)
		}

		return fmt.Errorf("telegram %s request failed: op=%s", method, urlErr.Op)
	}

	return fmt.Errorf("telegram %s request failed: %v", method, err)
}
