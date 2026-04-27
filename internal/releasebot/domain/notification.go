package domain

import "errors"

// BuildNotification is sent by the reusable build workflow after a successful image build.
type BuildNotification struct {
	Repository    string `json:"repository"`
	Ref           string `json:"ref"`
	Branch        string `json:"branch"`
	SHA           string `json:"sha"`
	Tag           string `json:"tag"`
	Actor         string `json:"actor"`
	CommitMessage string `json:"commit_message"`
	RunURL        string `json:"run_url"`
}

func (n BuildNotification) Validate() error {
	if n.Repository == "" {
		return errors.New("repository is required")
	}

	if n.Ref == "" {
		return errors.New("ref is required")
	}

	if n.Tag == "" {
		return errors.New("tag is required")
	}

	if n.Actor == "" {
		return errors.New("actor is required")
	}

	if n.RunURL == "" {
		return errors.New("run_url is required")
	}

	return nil
}
