package domain

import "testing"

func TestParseCallbackData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		data       string
		wantAction string
		wantID     string
		wantEnv    string
		wantErr    bool
	}{
		{
			name:       "release action",
			data:       "release:abc123",
			wantAction: ActionRelease,
			wantID:     "abc123",
		},
		{
			name:       "deploy action",
			data:       "deploy:abc123:prod",
			wantAction: ActionDeploy,
			wantID:     "abc123",
			wantEnv:    EnvironmentProd,
		},
		{
			name:    "unknown environment",
			data:    "deploy:abc123:stage",
			wantErr: true,
		},
		{
			name:    "malformed callback",
			data:    "release",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			callbackData, err := ParseCallbackData(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("parse callback: %v", err)
			}

			if callbackData.Action != tt.wantAction || callbackData.ReleaseID != tt.wantID ||
				callbackData.Environment != tt.wantEnv {
				t.Fatalf("unexpected callback parts: %+v", callbackData)
			}
		})
	}
}

func TestBuildNotificationValidate(t *testing.T) {
	t.Parallel()

	validNotification := BuildNotification{
		Repository:    "SOTBI-LLC/service",
		Ref:           "main",
		SHA:           "abc",
		Tag:           "v1.2.3",
		Actor:         "octocat",
		CommitMessage: "ship release",
		RunURL:        "https://github.com/SOTBI-LLC/service/actions/runs/1",
	}

	tests := []struct {
		name       string
		mutate     func(*BuildNotification)
		wantErrMsg string
	}{
		{
			name: "valid notification",
		},
		{
			name: "missing repository",
			mutate: func(notification *BuildNotification) {
				notification.Repository = ""
			},
			wantErrMsg: "repository is required",
		},
		{
			name: "missing ref",
			mutate: func(notification *BuildNotification) {
				notification.Ref = ""
			},
			wantErrMsg: "ref is required",
		},
		{
			name: "missing tag",
			mutate: func(notification *BuildNotification) {
				notification.Tag = ""
			},
			wantErrMsg: "tag is required",
		},
		{
			name: "missing actor",
			mutate: func(notification *BuildNotification) {
				notification.Actor = ""
			},
			wantErrMsg: "actor is required",
		},
		{
			name: "missing run url",
			mutate: func(notification *BuildNotification) {
				notification.RunURL = ""
			},
			wantErrMsg: "run_url is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			notification := validNotification
			if tt.mutate != nil {
				tt.mutate(&notification)
			}

			err := notification.Validate()
			if tt.wantErrMsg == "" {
				if err != nil {
					t.Fatalf("validate notification: %v", err)
				}

				return
			}

			if err == nil || err.Error() != tt.wantErrMsg {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
