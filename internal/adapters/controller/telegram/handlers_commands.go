package telegram

import (
	"context"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (c *Controller) start(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	userID := upd.Message.From.ID
	chatID := upd.Message.Chat.ID
	_ = c.form.Cancel(ctx, userID)

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
