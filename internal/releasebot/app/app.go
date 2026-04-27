package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SOTBI-LLC/gh-actions/internal/config"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/server"
)

const (
	defaultHTTPRequestTimeout = 45 * time.Second
	defaultShutdownTimeout    = 15 * time.Second
)

func Run() error {
	config, err := config.LoadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, *config, slog.Default())
}

func run(ctx context.Context, config config.Params, logger *slog.Logger) error {
	tr := &http.Transport{
		ForceAttemptHTTP2: true,
	}
	httpClient := &http.Client{Timeout: defaultHTTPRequestTimeout, Transport: tr}
	bot := server.New(config, logger, httpClient)
	mux := http.NewServeMux()
	bot.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:              config.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info(
		"starting release bot",
		"http_addr",
		config.HTTPAddr,
		"long_polling",
		config.EnableLongPolling,
		"telegram_api_base_url",
		config.TelegramAPIBaseURL,
		"github_api_base_url",
		config.GitHubAPIBaseURL,
		"workflow_file",
		config.WorkflowFile,
		"http_request_timeout",
		defaultHTTPRequestTimeout,
	)

	errCh := make(chan error, 2)

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	if config.EnableLongPolling {
		go func() {
			errCh <- bot.PollTelegramUpdates(ctx)
		}()
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		shutdownErr := httpServer.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			return errors.Join(err, shutdownErr)
		}

		return err
	}
}
