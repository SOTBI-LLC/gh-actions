// Package releasebot implements the Telegram release bot and GitHub Actions workflow dispatch.
package releasebot

import "github.com/SOTBI-LLC/gh-actions/internal/releasebot/app"

// Run starts the release bot service.
func Run() error {
	return app.Run()
}
