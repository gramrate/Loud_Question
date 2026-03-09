package telegram

import (
	"LoudQuestionBot/internal/domain/errorz"
	"context"
	"errors"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (c *Controller) start(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	userID := upd.Message.From.ID
	chatID := upd.Message.Chat.ID
	text := strings.TrimSpace(upd.Message.Text)
	profile := userProfileFromTelegramUser(*upd.Message.From)
	_ = c.form.Cancel(ctx, userID)

	if teamID, ok := parseStartJoinTeam(text); ok {
		if err := c.team.Join(ctx, teamID, userID, profile); err != nil {
			switch {
			case errors.Is(err, errorz.ErrNotFound):
				_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Команда не найдена"})
			case errors.Is(err, errorz.ErrAlreadyExists):
				_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы уже в этой команде"})
			case errors.Is(err, errorz.ErrConflict):
				_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы уже состоите в другой команде"})
			case errors.Is(err, errorz.ErrLimitExceeded):
				_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "В команде уже 10 участников"})
			default:
				_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось вступить в команду"})
			}
		} else {
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы вступили в команду"})
		}
	}

	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Добро пожаловать в Громкий вопрос",
		ReplyMarkup: c.mainMenu(userID),
	})
}

func (c *Controller) menu(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.form.Cancel(ctx, userID)

	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Главное меню",
		ReplyMarkup: c.mainMenu(userID),
	})
}

func (c *Controller) joinTeamByCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	userID := upd.Message.From.ID
	chatID := upd.Message.Chat.ID
	args := strings.Fields(strings.TrimSpace(upd.Message.Text))
	if len(args) != 2 {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Использование: /jointeam <uuid>",
		})
		return
	}

	teamID := args[1]
	if !isValidUUID(teamID) {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Неверный формат UUID"})
		return
	}
	err := c.team.Join(ctx, teamID, userID, userProfileFromTelegramUser(*upd.Message.From))
	if err != nil {
		switch {
		case errors.Is(err, errorz.ErrNotFound):
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Команда не найдена"})
		case errors.Is(err, errorz.ErrAlreadyExists):
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы уже в этой команде"})
		case errors.Is(err, errorz.ErrConflict):
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы уже состоите в другой команде"})
		case errors.Is(err, errorz.ErrLimitExceeded):
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "В команде уже 10 участников"})
		default:
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось вступить в команду"})
		}
		return
	}

	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Вы вступили в команду",
		ReplyMarkup: c.mainMenu(userID),
	})
}
