package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string, httpClient *http.Client) *Client {
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
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	var apiResponse APIResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return err
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices ||
		!apiResponse.OK {
		return fmt.Errorf(
			"telegram %s failed: status=%d description=%s",
			method,
			resp.StatusCode,
			apiResponse.Description,
		)
	}

	if result == nil || len(apiResponse.Result) == 0 {
		return nil
	}

	if err := json.Unmarshal(apiResponse.Result, result); err != nil {
		return err
	}

	return nil
}
