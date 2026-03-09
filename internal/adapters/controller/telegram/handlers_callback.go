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
	messageID := cb.Message.Message.ID
	data := cb.Data
	c.answerCallback(ctx, cb.ID, "")

	switch {
	case data == "menu":
		c.sendMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "play":
		c.sendNextQuestion(ctx, chatID, userID)
	case strings.HasPrefix(data, "ans:"):
		id, ok := parseStringPart(data, 1)
		if !ok || !isValidUUID(id) {
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
	case data == "team:menu":
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "team:create":
		_, err := c.team.Create(ctx, userID, userProfileFromTelegramUser(cb.From))
		if err != nil {
			switch {
			case errors.Is(err, errorz.ErrConflict):
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы уже состоите в команде"})
			default:
				log.Printf("team create: %v", err)
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось создать команду"})
			}
			return
		}
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "team:join:help":
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Введите код: /jointeam <uuid>"})
	case data == "team:leave":
		err := c.team.Leave(ctx, userID)
		if err != nil {
			if !errors.Is(err, errorz.ErrNotFound) {
				log.Printf("team leave: %v", err)
			}
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы не состоите в команде"})
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы вышли из команды"})
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "team:link":
		c.sendTeamInvite(ctx, chatID, userID)
	case data == "team:members":
		c.sendTeamMembers(ctx, chatID, userID)
	case data == "team:owner:list":
		c.sendTeamOwnerTransferMenu(ctx, chatID, userID, messageID)
	case strings.HasPrefix(data, "team:kick:"):
		memberID, ok := parseInt64Part(data, 2)
		if !ok {
			return
		}
		err := c.team.Kick(ctx, userID, memberID)
		if err != nil {
			switch {
			case errors.Is(err, errorz.ErrForbidden):
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Кикать участников может только создатель команды"})
			case errors.Is(err, errorz.ErrNotFound):
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Участник не найден"})
			default:
				log.Printf("team kick: %v", err)
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось кикнуть участника"})
			}
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Участник кикнут из команды"})
		c.sendTeamMembers(ctx, chatID, userID)
	case strings.HasPrefix(data, "team:owner:"):
		memberID, ok := parseInt64Part(data, 2)
		if !ok {
			return
		}
		err := c.team.TransferOwnership(ctx, userID, memberID)
		if err != nil {
			switch {
			case errors.Is(err, errorz.ErrForbidden):
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Передавать админа может только текущий создатель"})
			case errors.Is(err, errorz.ErrNotFound):
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Участник не найден в вашей команде"})
			default:
				log.Printf("team transfer ownership: %v", err)
				_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось передать админа"})
			}
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Админ передан"})
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "adm:menu":
		if !c.access.IsAdmin(userID) {
			_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Недостаточно прав"})
			return
		}
		c.sendAdminMenuWithMessage(ctx, chatID, messageID)
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
		qid := parts[2]
		page, err := strconv.Atoi(parts[3])
		if err != nil || !isValidUUID(qid) {
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
		qid := parts[2]
		page, err := strconv.Atoi(parts[3])
		if err != nil || !isValidUUID(qid) {
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
		if !isValidUUID(qid) {
			return
		}
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
		qid := parts[2]
		page, err := strconv.Atoi(parts[3])
		if err != nil || !isValidUUID(qid) {
			return
		}
		err = c.admin.DeleteQuestion(ctx, userID, qid)
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
		c.sendAdminMenuWithMessage(ctx, chatID, messageID)
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
	teamID := ""
	if t, ok, err := c.team.GetByUserID(ctx, userID); err == nil && ok {
		teamID = t.ID
	}
	q, err := c.game.NextQuestion(ctx, userID, teamID)
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
			{{Text: "Показать ответ", CallbackData: fmt.Sprintf("ans:%s", q.ID)}},
			{{Text: "Следующий вопрос", CallbackData: "play"}},
		}},
	})
}
