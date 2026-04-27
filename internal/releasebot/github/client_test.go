package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SOTBI-LLC/gh-actions/internal/config"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
)

func TestDispatchWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		notification domain.BuildNotification
		environment  string
		status       int
		wantErr      bool
		wantPath     string
		wantRef      string
		wantTag      string
		wantEnv      string
	}{
		{
			name: "dispatches workflow",
			notification: domain.BuildNotification{
				Repository: "/SOTBI-LLC/service/",
				Ref:        "main",
				Tag:        "v1.2.3",
			},
			environment: domain.EnvironmentProd,
			status:      http.StatusNoContent,
			wantPath:    "/repos/SOTBI-LLC/service/actions/workflows/restart.yaml/dispatches",
			wantRef:     "main",
			wantTag:     "v1.2.3",
			wantEnv:     domain.EnvironmentProd,
		},
		{
			name: "returns github error",
			notification: domain.BuildNotification{
				Repository: "SOTBI-LLC/service",
				Ref:        "main",
				Tag:        "v1.2.3",
			},
			environment: domain.EnvironmentDev,
			status:      http.StatusBadRequest,
			wantErr:     true,
			wantPath:    "/repos/SOTBI-LLC/service/actions/workflows/restart.yaml/dispatches",
			wantRef:     "main",
			wantTag:     "v1.2.3",
			wantEnv:     domain.EnvironmentDev,
		},
		{
			name: "rejects repository without owner",
			notification: domain.BuildNotification{
				Repository: "service",
				Ref:        "main",
				Tag:        "v1.2.3",
			},
			environment: domain.EnvironmentDev,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				gotPath    string
				gotAuth    string
				gotPayload struct {
					Ref    string            `json:"ref"`
					Inputs map[string]string `json:"inputs"`
				}
			)

			githubAPI := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotPath = r.URL.Path

					gotAuth = r.Header.Get("Authorization")
					if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
						t.Fatalf("decode github dispatch: %v", err)
					}

					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte("github error"))
				}),
			)
			defer githubAPI.Close()

			client := NewClient(
				githubAPI.URL,
				"github-token",
				config.DefaultWorkflowFile,
				githubAPI.Client(),
			)

			err := client.DispatchWorkflow(context.Background(), tt.notification, tt.environment)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected dispatch error")
				}

				if tt.wantPath == "" && gotPath != "" {
					t.Fatalf("github API should not have been called, got path %s", gotPath)
				}

				return
			}

			if err != nil {
				t.Fatalf("dispatch workflow: %v", err)
			}

			if gotPath != tt.wantPath {
				t.Fatalf("unexpected github path: %s", gotPath)
			}

			if gotAuth != "Bearer github-token" {
				t.Fatalf("unexpected authorization header: %s", gotAuth)
			}

			if gotPayload.Ref != tt.wantRef || gotPayload.Inputs["tag"] != tt.wantTag ||
				gotPayload.Inputs["environment"] != tt.wantEnv {
				t.Fatalf("unexpected github payload: %+v", gotPayload)
			}
		})
	}
}
