package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/telegram"
)

const defaultPollTimeout = 30

func (s *Server) PollTelegramUpdates(ctx context.Context) error {
	var offset int64

	for {
		updates, err := s.telegram.GetUpdates(ctx, offset, defaultPollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			s.logger.Warn("failed to poll telegram updates", "error", err)
			time.Sleep(time.Second)

			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if err := s.HandleTelegramUpdate(ctx, update); err != nil {
				s.logger.Warn(
					"failed to handle telegram update",
					"update_id",
					update.UpdateID,
					"error",
					err,
				)
			}
		}
	}
}

func (s *Server) HandleTelegramUpdate(ctx context.Context, update telegram.Update) error {
	if update.CallbackQuery == nil {
		return nil
	}

	callback := update.CallbackQuery
	if !s.isAllowedUser(callback.From.ID) {
		return errors.Join(
			domain.ErrUnauthorized,
			s.telegram.AnswerCallback(
				ctx,
				callback.ID,
				"You are not allowed to release this service",
				true,
			),
		)
	}

	callbackData, err := domain.ParseCallbackData(callback.Data)
	if err != nil {
		return err
	}

	switch callbackData.Action {
	case domain.ActionRelease:
		return s.handleReleaseCallback(ctx, callback, callbackData.ReleaseID)
	case domain.ActionDeploy:
		return s.handleDeployCallback(
			ctx,
			callback,
			callbackData.ReleaseID,
			callbackData.Environment,
		)
	default:
		return fmt.Errorf("%w: unknown action %q", domain.ErrBadCallback, callbackData.Action)
	}
}

func (s *Server) handleReleaseCallback(
	ctx context.Context,
	callback *telegram.CallbackQuery,
	releaseID string,
) error {
	if _, ok := s.releases.Get(releaseID); !ok {
		return s.telegram.AnswerCallback(ctx, callback.ID, "Release context has expired", true)
	}

	if callback.Message == nil {
		return fmt.Errorf("%w: callback message is missing", domain.ErrBadCallback)
	}

	if err := s.telegram.EditMessageReplyMarkup(
		ctx,
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		telegram.EnvironmentKeyboard(releaseID)); err != nil {
		return err
	}

	return s.telegram.AnswerCallback(ctx, callback.ID, "Choose environment", false)
}

func (s *Server) handleDeployCallback(
	ctx context.Context,
	callback *telegram.CallbackQuery,
	releaseID, environment string,
) error {
	notification, ok := s.releases.Get(releaseID)
	if !ok {
		return s.telegram.AnswerCallback(ctx, callback.ID, "Release context has expired", true)
	}

	if err := s.github.DispatchWorkflow(ctx, notification, environment); err != nil {
		_ = s.telegram.AnswerCallback(ctx, callback.ID, "Failed to start deploy", true)

		return err
	}

	if callback.Message != nil {
		_ = s.telegram.EditMessageReplyMarkup(
			ctx,
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			telegram.DeployedKeyboard(environment),
		)
	}

	return s.telegram.AnswerCallback(ctx, callback.ID, "Deploy started for "+environment, false)
}
