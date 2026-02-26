package telegram

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/schema"
	gamesvc "LoudQuestionBot/internal/domain/service/game"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (c *Controller) handleCallback(ctx context.Context, upd *models.Update) {
	cb := upd.CallbackQuery
	if cb == nil {
		return
	}
	userID := cb.From.ID
	chatID := cb.Message.Message.Chat.ID
	data := cb.Data
	c.answerCallback(ctx, cb.ID, "")

	switch {
	case data == "menu":
		c.sendMenu(ctx, chatID, userID)
	case data == "play":
		c.sendNextQuestion(ctx, chatID, userID)
	case strings.HasPrefix(data, "ans:"):
		id, ok := parseInt64Part(data, 1)
		if !ok {
			return
		}
		answer, err := c.game.AnswerByQuestionID(ctx, id)
		if err != nil {
			if errors.Is(err, errorz.ErrNotFound) {
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос больше недоступен"})
				return
			}
			log.Printf("answer by id: %v", err)
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ответ: " + answer})
	case data == "adm:menu":
		if !c.access.IsAdmin(userID) {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Недостаточно прав"})
			return
		}
		c.sendAdminMenu(ctx, chatID)
	case data == "adm:add":
		if !c.access.IsAdmin(userID) {
			return
		}
		_ = c.form.StartCreate(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Напишите вопрос"})
	case strings.HasPrefix(data, "adm:list:"):
		if !c.access.IsAdmin(userID) {
			return
		}
		page, ok := parseIntPart(data, 2)
		if !ok {
			return
		}
		c.sendMyQuestions(ctx, chatID, userID, page)
	case strings.HasPrefix(data, "adm:open:"):
		if !c.access.IsAdmin(userID) {
			return
		}
		parts := strings.Split(data, ":")
		if len(parts) < 4 {
			return
		}
		qid, err1 := strconv.ParseInt(parts[2], 10, 64)
		page, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			return
		}
		c.sendQuestionCard(ctx, chatID, userID, qid, page)
	case strings.HasPrefix(data, "adm:edit:"):
		if !c.access.IsAdmin(userID) {
			return
		}
		parts := strings.Split(data, ":")
		if len(parts) < 4 {
			return
		}
		qid, err1 := strconv.ParseInt(parts[2], 10, 64)
		page, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			return
		}
		q, err := c.admin.GetQuestion(ctx, qid)
		if err != nil {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос больше недоступен"})
			return
		}
		if q.AuthorID != userID || q.Status != schema.QuestionStatusActive {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Можно редактировать только свои активные вопросы"})
			return
		}
		_ = c.form.StartEdit(ctx, userID, q.ID, page, schema.QuestionDraft{QuestionText: q.QuestionText, AnswerText: q.AnswerText})
		c.sendChooseField(ctx, chatID)
	case strings.HasPrefix(data, "adm:delask:"):
		if !c.access.IsAdmin(userID) {
			return
		}
		parts := strings.Split(data, ":")
		if len(parts) < 4 {
			return
		}
		qid := parts[2]
		page := parts[3]
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Точно удалить вопрос? Он исчезнет у всех игроков.",
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "✅ Да, удалить", CallbackData: "adm:del:" + qid + ":" + page}},
				{{Text: "❌ Нет, отмена", CallbackData: "adm:open:" + qid + ":" + page}},
			}},
		})
	case strings.HasPrefix(data, "adm:del:"):
		if !c.access.IsAdmin(userID) {
			return
		}
		parts := strings.Split(data, ":")
		if len(parts) < 4 {
			return
		}
		qid, err1 := strconv.ParseInt(parts[2], 10, 64)
		page, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			return
		}
		err := c.admin.DeleteQuestion(ctx, userID, qid)
		if err != nil {
			if errors.Is(err, errorz.ErrForbidden) {
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Можно удалять только свои вопросы"})
				return
			}
			log.Printf("delete question: %v", err)
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "🗑 Удалено"})
		c.sendMyQuestions(ctx, chatID, userID, page)
	case data == "frm:x":
		_ = c.form.Cancel(ctx, userID)
		c.sendAdminMenu(ctx, chatID)
	case data == "frm:e":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Форма устарела, начните заново"})
			return
		}
		state.Step = schema.FormStepChooseField
		_ = c.form.Save(ctx, userID, state)
		c.sendChooseField(ctx, chatID)
	case data == "frm:b":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Форма устарела, начните заново"})
			return
		}
		state.Step = schema.FormStepPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendDraftPreview(ctx, chatID, state)
	case data == "frm:f:q" || data == "frm:f:a":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Форма устарела, начните заново"})
			return
		}
		switch data {
		case "frm:f:q":
			state.Field = schema.FormFieldQuestion
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Введите новый текст вопроса"})
		case "frm:f:a":
			state.Field = schema.FormFieldAnswer
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Введите новый текст ответа"})
		}
		state.Step = schema.FormStepEditInput
		_ = c.form.Save(ctx, userID, state)
	case data == "frm:c":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Форма устарела, начните заново"})
			return
		}
		if state.Mode != schema.FormModeCreate {
			return
		}
		_, err = c.admin.CreateQuestion(ctx, userID, state.Draft)
		if err != nil {
			log.Printf("create question: %v", err)
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось сохранить вопрос"})
			return
		}
		_ = c.form.Cancel(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "✅ Вопрос добавлен"})
		c.sendAdminMenu(ctx, chatID)
	case data == "frm:s":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Форма устарела, начните заново"})
			return
		}
		if state.Mode != schema.FormModeEdit {
			return
		}
		q, err := c.admin.UpdateQuestion(ctx, userID, state.QuestionID, state.Draft)
		if err != nil {
			if errors.Is(err, errorz.ErrForbidden) {
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Можно редактировать только свои активные вопросы"})
				return
			}
			log.Printf("update question: %v", err)
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось обновить вопрос"})
			return
		}
		_ = c.form.Cancel(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "✅ Обновлено"})
		c.sendQuestionCardWithEntity(ctx, chatID, q, state.Page)
	}
}

func (c *Controller) sendNextQuestion(ctx context.Context, chatID, userID int64) {
	q, err := c.game.NextQuestion(ctx, userID)
	if err != nil {
		if errors.Is(err, gamesvc.ErrNoNewQuestions) {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Нет новых вопросов", ReplyMarkup: c.mainMenu(userID)})
			return
		}
		log.Printf("next question: %v", err)
		return
	}

	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "Вопрос:\n" + q.QuestionText,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Показать ответ", CallbackData: fmt.Sprintf("ans:%d", q.ID)}},
			{{Text: "Следующий вопрос", CallbackData: "play"}},
		}},
	})
}
