package domain

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ActionRelease   = "release"
	ActionDeploy    = "deploy"
	ActionNoop      = "noop"
	EnvironmentDev  = "dev"
	EnvironmentProd = "prod"
)

var (
	ErrUnauthorized = errors.New("unauthorized telegram user")
	ErrBadCallback  = errors.New("bad callback data")
)

type CallbackData struct {
	Action      string
	ReleaseID   string
	Environment string
}

func ParseCallbackData(data string) (CallbackData, error) {
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return CallbackData{}, fmt.Errorf("%w: %q", ErrBadCallback, data)
	}

	switch parts[0] {
	case ActionRelease:
		if len(parts) != 2 {
			return CallbackData{}, fmt.Errorf("%w: %q", ErrBadCallback, data)
		}

		return CallbackData{Action: parts[0], ReleaseID: parts[1]}, nil
	case ActionDeploy:
		if len(parts) != 3 {
			return CallbackData{}, fmt.Errorf("%w: %q", ErrBadCallback, data)
		}

		if parts[2] != EnvironmentDev && parts[2] != EnvironmentProd {
			return CallbackData{}, fmt.Errorf(
				"%w: unknown environment %q",
				ErrBadCallback,
				parts[2],
			)
		}

		return CallbackData{Action: parts[0], ReleaseID: parts[1], Environment: parts[2]}, nil
	default:
		return CallbackData{}, fmt.Errorf("%w: unknown action %q", ErrBadCallback, parts[0])
	}
}
