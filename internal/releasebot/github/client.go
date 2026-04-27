package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
)

type Client struct {
	baseURL      string
	token        string
	workflowFile string
	http         *http.Client
}

func NewClient(baseURL, token, workflowFile string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		token:        token,
		workflowFile: workflowFile,
		http:         httpClient,
	}
}

func (c *Client) DispatchWorkflow(
	ctx context.Context,
	notification domain.BuildNotification,
	environment string,
) error {
	repo := strings.Trim(notification.Repository, "/")
	if !strings.Contains(repo, "/") {
		return fmt.Errorf(
			"repository must be owner/name: %s",
			notification.Repository,
		)
	}

	payload := map[string]any{
		"ref": notification.Ref,
		"inputs": map[string]string{
			"tag":         notification.Tag,
			"environment": environment,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/repos/%s/actions/workflows/%s/dispatches",
		c.baseURL,
		repo,
		c.workflowFile,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNoContent {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

		return fmt.Errorf(
			"github workflow dispatch failed: status=%d body=%s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}
