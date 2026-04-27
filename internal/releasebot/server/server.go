package server

import (
	"log/slog"
	"net/http"

	"github.com/SOTBI-LLC/gh-actions/internal/config"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/github"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/store"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/telegram"
)

type Server struct {
	config   config.Params
	releases *store.Store
	telegram *telegram.Client
	github   *github.Client
	logger   *slog.Logger
}

func New(config config.Params, logger *slog.Logger, httpClient *http.Client) *Server {
	return &Server{
		config:   config,
		releases: store.New(),
		telegram: telegram.NewClient(
			config.TelegramAPIBaseURL,
			config.TelegramBotToken,
			httpClient,
		),
		github: github.NewClient(
			config.GitHubAPIBaseURL,
			config.GitHubToken,
			config.WorkflowFile,
			httpClient,
		),
		logger: logger,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /build-notifications", s.handleBuildNotification)
	mux.HandleFunc("POST /telegram/webhook", s.handleTelegramWebhook)
}

func (s *Server) isAllowedUser(userID int64) bool {
	_, ok := s.config.TelegramAllowedUserIDs[userID]

	return ok
}
