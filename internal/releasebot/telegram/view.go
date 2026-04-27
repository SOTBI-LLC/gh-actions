package telegram

import (
	"strings"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
)

func ReleaseKeyboard(releaseID string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         "release",
					CallbackData: domain.ActionRelease + ":" + releaseID,
				},
			},
		},
	}
}

func EnvironmentKeyboard(releaseID string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         domain.EnvironmentDev,
					CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentDev,
				},
				{
					Text:         domain.EnvironmentProd,
					CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + domain.EnvironmentProd,
				},
			},
		},
	}
}

// PostDeployEnvironmentKeyboard keeps dev/prod available so the other environment
// can be deployed after the first dispatch. Completed environments use a no-op button.
func PostDeployEnvironmentKeyboard(releaseID string, devDone, prodDone bool) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				environmentButton(releaseID, domain.EnvironmentDev, devDone),
				environmentButton(releaseID, domain.EnvironmentProd, prodDone),
			},
		},
	}
}

func environmentButton(releaseID, environment string, done bool) InlineKeyboardButton {
	if done {
		return InlineKeyboardButton{
			Text:         "✓ " + environment,
			CallbackData: domain.ActionNoop,
		}
	}

	return InlineKeyboardButton{
		Text:         environment,
		CallbackData: domain.ActionDeploy + ":" + releaseID + ":" + environment,
	}
}

func RenderBuildMessage(notification domain.BuildNotification) string {
	branch := notification.Branch
	if branch == "" {
		branch = notification.Ref
	}

	return strings.Join([]string{
		"🏗️: " + escapeMarkdown(notification.Actor) + " built image.",
		"💬: " + escapeMarkdown(notification.CommitMessage),
		"🌴: " + escapeMarkdown(branch),
		"🔖: " + escapeMarkdown(notification.Tag),
		"👀 changes: https://github.com/" + notification.Repository + "/commit/" + notification.SHA,
		"📋: " + notification.RunURL,
	}, "\n")
}

func escapeMarkdown(value string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"`", "\\`",
		"[", "\\[",
	)

	return replacer.Replace(value)
}
