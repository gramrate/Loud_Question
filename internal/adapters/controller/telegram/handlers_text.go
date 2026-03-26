package telegram

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"log"
	"strings"
	"unicode/utf8"

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
		if utf8.RuneCountInString(text) > 250 {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос не должен быть длиннее 250 символов"})
			return
		}
		state.Draft.QuestionText = text
		state.Step = schema.FormStepAnswer
		_ = c.form.Save(ctx, userID, state)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Напишите ответ"})
	case schema.FormStepAnswer:
		if utf8.RuneCountInString(text) > 250 {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ответ не должен быть длиннее 250 символов"})
			return
		}
		state.Draft.AnswerText = text
		state.Step = schema.FormStepPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendDraftPreview(ctx, chatID, state)
	case schema.FormStepEditInput:
		switch state.Field {
		case schema.FormFieldQuestion:
			if utf8.RuneCountInString(text) > 250 {
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос не должен быть длиннее 250 символов"})
				return
			}
			state.Draft.QuestionText = text
		case schema.FormFieldAnswer:
			if utf8.RuneCountInString(text) > 250 {
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ответ не должен быть длиннее 250 символов"})
				return
			}
			state.Draft.AnswerText = text
		}
		state.Step = schema.FormStepPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendDraftPreview(ctx, chatID, state)
	case schema.FormStepPoolInput:
		items, err := parsePoolQuestions(text)
		if err != nil {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ошибка парсинга: " + err.Error()})
			return
		}
		state.Step = schema.FormStepPoolPreview
		state.PoolItems = items
		state.PoolIndex = 0
		state.PoolSaved = 0
		_ = c.form.Save(ctx, userID, state)
		c.sendPoolPreview(ctx, chatID, state)
	case schema.FormStepPoolEditQ:
		if utf8.RuneCountInString(text) > 250 {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос не должен быть длиннее 250 символов"})
			return
		}
		if strings.TrimSpace(text) == "" {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос не может быть пустым"})
			return
		}
		state.Draft.QuestionText = strings.TrimSpace(text)
		state.Step = schema.FormStepPoolEditA
		_ = c.form.Save(ctx, userID, state)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Введите ответ"})
	case schema.FormStepPoolEditA:
		if utf8.RuneCountInString(text) > 250 {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ответ не должен быть длиннее 250 символов"})
			return
		}
		if strings.TrimSpace(text) == "" {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ответ не может быть пустым"})
			return
		}
		if state.PoolIndex < 0 || state.PoolIndex >= len(state.PoolItems) {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Форма устарела, начните заново"})
			_ = c.form.Cancel(ctx, userID)
			return
		}
		state.Draft.AnswerText = strings.TrimSpace(text)
		state.PoolItems[state.PoolIndex] = schema.QuestionDraft{
			QuestionText: state.Draft.QuestionText,
			AnswerText:   state.Draft.AnswerText,
		}
		state.Step = schema.FormStepPoolPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendPoolPreview(ctx, chatID, state)
	default:
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Используйте кнопки под сообщением"})
	}
}
