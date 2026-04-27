package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/httpjson"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/security"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/telegram"
)

const defaultCallbackTimeout = 10 * time.Second

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBuildNotification(w http.ResponseWriter, r *http.Request) {
	if !security.HasSharedSecret(
		r.Header.Get("X-Releasebot-Secret"),
		s.config.BuildNotificationSecret,
	) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)

		return
	}

	var notification domain.BuildNotification
	if err := httpjson.DecodeBody(r.Body, &notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if err := notification.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	releaseID, err := s.releases.Create(notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultCallbackTimeout)
	defer cancel()

	if err := s.telegram.SendMessage(
		ctx, s.config.TelegramChatID, telegram.RenderBuildMessage(notification),
		telegram.ReleaseKeyboard(releaseID)); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)

		return
	}

	httpjson.Write(w, http.StatusAccepted, map[string]string{"release_id": releaseID})
}

func (s *Server) handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if s.config.TelegramWebhookSecretToken != "" &&
		!security.HasSharedSecret(
			r.Header.Get("X-Telegram-Bot-Api-Secret-Token"),
			s.config.TelegramWebhookSecretToken,
		) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)

		return
	}

	var update telegram.Update
	if err := httpjson.DecodeBody(r.Body, &update, false); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if err := s.HandleTelegramUpdate(r.Context(), update); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrUnauthorized) {
			status = http.StatusForbidden
		}

		if errors.Is(err, domain.ErrBadCallback) {
			status = http.StatusBadRequest
		}

		http.Error(w, err.Error(), status)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
