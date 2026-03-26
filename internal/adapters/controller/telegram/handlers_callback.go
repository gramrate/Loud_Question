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
	answered := false
	ack := func(text string, showAlert bool) {
		if answered {
			return
		}
		c.answerCallback(ctx, cb.ID, truncateForAlert(text), showAlert)
		answered = true
	}
	defer func() {
		if !answered {
			ack("", false)
		}
	}()

	switch {
	case data == "menu":
		c.sendMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "profile:menu":
		c.sendProfileMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "play":
		c.sendNextQuestionFromCallback(ctx, chatID, userID, ack)
	case strings.HasPrefix(data, "ans:"):
		id, ok := parseStringPart(data, 1)
		if !ok || !isValidUUID(id) {
			return
		}
		answer, err := c.game.AnswerByQuestionID(ctx, id)
		if err != nil {
			if errors.Is(err, errorz.ErrNotFound) {
				ack("Вопрос больше недоступен", true)
				return
			}
			log.Printf("answer by id: %v", err)
			ack("Ошибка. Попробуйте позже", true)
			return
		}
		if err := c.game.MarkAnsweredByUser(ctx, userID, id); err != nil {
			log.Printf("mark answered by user: %v", err)
		}
		ack("Ответ: "+answer, true)
	case data == "team:menu":
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "team:create":
		_, err := c.team.Create(ctx, userID, userProfileFromTelegramUser(cb.From))
		if err != nil {
			switch {
			case errors.Is(err, errorz.ErrConflict):
				ack("Вы уже состоите в команде", true)
			default:
				log.Printf("team create: %v", err)
				ack("Не удалось создать команду", true)
			}
			return
		}
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "team:join:help":
		ack("Введите код: /jointeam <uuid>", true)
	case data == "team:leave":
		err := c.team.Leave(ctx, userID)
		if err != nil {
			if !errors.Is(err, errorz.ErrNotFound) {
				log.Printf("team leave: %v", err)
			}
			ack("Вы не состоите в команде", true)
			return
		}
		ack("Вы вышли из команды", true)
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "team:link":
		c.sendTeamInvite(ctx, chatID, userID)
	case data == "team:members":
		c.sendTeamMembersWithMessage(ctx, chatID, userID, messageID)
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
				ack("Кикать может только создатель", true)
			case errors.Is(err, errorz.ErrNotFound):
				ack("Участник не найден", true)
			default:
				log.Printf("team kick: %v", err)
				ack("Не удалось кикнуть участника", true)
			}
			return
		}
		ack("Участник кикнут", true)
		c.sendTeamMembersWithMessage(ctx, chatID, userID, messageID)
	case strings.HasPrefix(data, "team:owner:"):
		memberID, ok := parseInt64Part(data, 2)
		if !ok {
			return
		}
		err := c.team.TransferOwnership(ctx, userID, memberID)
		if err != nil {
			switch {
			case errors.Is(err, errorz.ErrForbidden):
				ack("Передавать может только создатель", true)
			case errors.Is(err, errorz.ErrNotFound):
				ack("Участник не найден в команде", true)
			default:
				log.Printf("team transfer ownership: %v", err)
				ack("Не удалось передать админа", true)
			}
			return
		}
		ack("Админ передан", true)
		c.sendTeamMenuWithMessage(ctx, chatID, userID, messageID)
	case data == "adm:menu":
		if !c.access.IsAdmin(userID) {
			ack("Недостаточно прав", true)
			return
		}
		c.sendAdminMenuWithMessage(ctx, chatID, messageID)
	case data == "adm:add":
		if !c.access.IsAdmin(userID) {
			return
		}
		_ = c.form.StartCreate(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Напишите вопрос"})
	case data == "adm:pool":
		if !c.access.IsAdmin(userID) {
			return
		}
		_ = c.form.StartPoolCreate(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Отправьте пулл вопросов (до 25) в формате:\n[2+2]-[4]\n[4+2]-[6]\n\nДля экстренной остановки: /stop",
		})
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
			ack("Вопрос больше недоступен", true)
			return
		}
		if q.AuthorID != userID || q.Status != schema.QuestionStatusActive {
			ack("Можно редактировать только свои", true)
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
				ack("Можно удалять только свои", true)
				return
			}
			log.Printf("delete question: %v", err)
			ack("Не удалось удалить вопрос", true)
			return
		}
		ack("Удалено", true)
		c.sendMyQuestions(ctx, chatID, userID, page)
	case data == "frm:x":
		_ = c.form.Cancel(ctx, userID)
		c.sendAdminMenuWithMessage(ctx, chatID, messageID)
	case data == "frm:e":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			ack("Форма устарела, начните заново", true)
			return
		}
		state.Step = schema.FormStepChooseField
		_ = c.form.Save(ctx, userID, state)
		c.sendChooseField(ctx, chatID)
	case data == "frm:b":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			ack("Форма устарела, начните заново", true)
			return
		}
		state.Step = schema.FormStepPreview
		_ = c.form.Save(ctx, userID, state)
		c.sendDraftPreview(ctx, chatID, state)
	case data == "frm:f:q" || data == "frm:f:a":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			ack("Форма устарела, начните заново", true)
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
			ack("Форма устарела, начните заново", true)
			return
		}
		if state.Mode != schema.FormModeCreate {
			return
		}
		_, err = c.admin.CreateQuestion(ctx, userID, state.Draft)
		if err != nil {
			if errors.Is(err, errorz.ErrLimitExceeded) {
				ack("Лимит 200 символов на вопрос и ответ", true)
				return
			}
			log.Printf("create question: %v", err)
			ack("Не удалось сохранить вопрос", true)
			return
		}
		_ = c.form.Cancel(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "✅ Вопрос добавлен"})
		c.sendAdminMenu(ctx, chatID)
	case data == "frm:p:e":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok || state.Step != schema.FormStepPoolPreview {
			ack("Форма устарела, начните заново", true)
			return
		}
		state.Step = schema.FormStepPoolEditQ
		_ = c.form.Save(ctx, userID, state)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите вопрос",
		})
	case data == "frm:p:x":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok || state.Step != schema.FormStepPoolPreview {
			ack("Форма устарела, начните заново", true)
			return
		}
		state.PoolIndex++
		if state.PoolIndex >= len(state.PoolItems) {
			_ = c.form.Cancel(ctx, userID)
			ack(fmt.Sprintf("Пулл завершен. Добавлено: %d из %d", state.PoolSaved, len(state.PoolItems)), true)
			c.sendAdminMenu(ctx, chatID)
			return
		}
		_ = c.form.Save(ctx, userID, state)
		c.sendPoolPreview(ctx, chatID, state)
	case data == "frm:p:c":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok || state.Step != schema.FormStepPoolPreview {
			ack("Форма устарела, начните заново", true)
			return
		}
		if state.PoolIndex < 0 || state.PoolIndex >= len(state.PoolItems) {
			ack("Форма устарела, начните заново", true)
			_ = c.form.Cancel(ctx, userID)
			return
		}
		_, err = c.admin.CreateQuestion(ctx, userID, state.PoolItems[state.PoolIndex])
		if err != nil {
			if errors.Is(err, errorz.ErrLimitExceeded) {
				ack("Лимит 200 символов на вопрос и ответ", true)
			} else {
				log.Printf("create pool question: %v", err)
				ack("Не удалось сохранить вопрос", true)
			}
			return
		}
		state.PoolSaved++
		state.PoolIndex++
		if state.PoolIndex >= len(state.PoolItems) {
			_ = c.form.Cancel(ctx, userID)
			ack(fmt.Sprintf("Пулл завершен. Добавлено: %d из %d", state.PoolSaved, len(state.PoolItems)), true)
			c.sendAdminMenu(ctx, chatID)
			return
		}
		_ = c.form.Save(ctx, userID, state)
		c.sendPoolPreview(ctx, chatID, state)
	case data == "frm:s":
		state, ok, err := c.form.Get(ctx, userID)
		if err != nil || !ok {
			ack("Форма устарела, начните заново", true)
			return
		}
		if state.Mode != schema.FormModeEdit {
			return
		}
		q, err := c.admin.UpdateQuestion(ctx, userID, state.QuestionID, state.Draft)
		if err != nil {
			if errors.Is(err, errorz.ErrForbidden) {
				ack("Можно редактировать только свои", true)
				return
			}
			if errors.Is(err, errorz.ErrLimitExceeded) {
				ack("Лимит 200 символов на вопрос и ответ", true)
				return
			}
			log.Printf("update question: %v", err)
			ack("Не удалось обновить вопрос", true)
			return
		}
		_ = c.form.Cancel(ctx, userID)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "✅ Обновлено"})
		c.sendQuestionCardWithEntity(ctx, chatID, q, state.Page)
	}
}

func (c *Controller) sendNextQuestion(ctx context.Context, chatID, userID int64) {
	q, err := c.nextQuestion(ctx, userID)
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

func (c *Controller) sendNextQuestionFromCallback(ctx context.Context, chatID, userID int64, ack func(string, bool)) {
	q, err := c.nextQuestion(ctx, userID)
	if err != nil {
		if errors.Is(err, gamesvc.ErrNoNewQuestions) {
			ack("Нет новых вопросов", true)
			return
		}
		log.Printf("next question: %v", err)
		ack("Ошибка. Попробуйте позже", true)
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

func (c *Controller) nextQuestion(ctx context.Context, userID int64) (schema.Question, error) {
	teamID := ""
	if t, ok, err := c.team.GetByUserID(ctx, userID); err == nil && ok {
		teamID = t.ID
	}
	return c.game.NextQuestion(ctx, userID, teamID)
}
