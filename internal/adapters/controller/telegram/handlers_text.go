package telegram

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"log"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (c *Controller) handleText(ctx context.Context, upd *models.Update) {
	msg := upd.Message
	if msg == nil || msg.From == nil {
		return
	}
	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	if strings.HasPrefix(text, "/") {
		if text == "/start" || text == "/menu" {
			_ = c.form.Cancel(ctx, userID)
		}
		return
	}

	state, ok, err := c.form.Get(ctx, userID)
	if err != nil {
		log.Printf("load form state: %v", err)
		return
	}
	if !ok {
		if text == "Играть" {
			c.sendNextQuestion(ctx, chatID, userID)
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Используйте /menu"})
		return
	}

	switch state.Step {
	case schema.FormStepQuestion:
		state.Draft.QuestionText = text
		state.Step = schema.FormStepAnswer
		_ = c.form.Save(ctx, userID, state)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Напишите ответ"})
	case schema.FormStepAnswer:
		state.Draft.AnswerText = text
		state.Step = schema.FormStepPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendDraftPreview(ctx, chatID, state)
	case schema.FormStepEditInput:
		switch state.Field {
		case schema.FormFieldQuestion:
			state.Draft.QuestionText = text
		case schema.FormFieldAnswer:
			state.Draft.AnswerText = text
		}
		state.Step = schema.FormStepPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendDraftPreview(ctx, chatID, state)
	default:
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Используйте кнопки под сообщением"})
	}
}
