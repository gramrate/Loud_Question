package telegram

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/schema"
	adminsvc "LoudQuestionBot/internal/domain/service/admin"
	"LoudQuestionBot/internal/domain/service/access"
	"LoudQuestionBot/internal/domain/service/form"
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

const pageSize = 10

type Runner struct {
	bot *tgbot.Bot
}

type Controller struct {
	bot    *tgbot.Bot
	access *access.Service
	game   *gamesvc.Service
	admin  *adminsvc.Service
	form   *form.Service
}

func New(token string, accessSvc *access.Service, gameSvc *gamesvc.Service, adminSvc *adminsvc.Service, formSvc *form.Service) (*Runner, error) {
	ctrl := &Controller{access: accessSvc, game: gameSvc, admin: adminSvc, form: formSvc}

	b, err := tgbot.New(token, tgbot.WithDefaultHandler(ctrl.defaultHandler))
	if err != nil {
		return nil, err
	}
	ctrl.bot = b

	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, ctrl.start)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/menu", tgbot.MatchTypeExact, ctrl.menu)

	return &Runner{bot: b}, nil
}

func (r *Runner) Start(ctx context.Context) {
	log.Println("telegram bot started")
	r.bot.Start(ctx)
}

func (c *Controller) start(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	userID := upd.Message.From.ID
	chatID := upd.Message.Chat.ID
	_ = c.form.Cancel(ctx, userID)

	text := "Добро пожаловать в Громкий вопрос"
	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
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

func (c *Controller) defaultHandler(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	switch {
	case upd.CallbackQuery != nil:
		c.handleCallback(ctx, upd)
	case upd.Message != nil && upd.Message.Text != "":
		c.handleText(ctx, upd)
	}
}

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

func (c *Controller) mainMenu(userID int64) *models.InlineKeyboardMarkup {
	rows := [][]models.InlineKeyboardButton{
		{{Text: "Играть", CallbackData: "play"}},
	}
	if c.access.IsAdmin(userID) {
		rows = append(rows, []models.InlineKeyboardButton{{Text: "Админка", CallbackData: "adm:menu"}})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (c *Controller) sendAdminMenu(ctx context.Context, chatID int64) {
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "Админ-панель",
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "➕ Добавить вопрос", CallbackData: "adm:add"}},
			{{Text: "📋 Мои вопросы", CallbackData: "adm:list:1"}},
			{{Text: "⬅ Назад", CallbackData: "menu"}},
		}},
	})
}

func (c *Controller) sendMenu(ctx context.Context, chatID, userID int64) {
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Главное меню",
		ReplyMarkup: c.mainMenu(userID),
	})
}

func (c *Controller) sendMyQuestions(ctx context.Context, chatID, userID int64, page int) {
	if page < 1 {
		page = 1
	}
	res, err := c.admin.MyQuestions(ctx, userID, page, pageSize)
	if err != nil {
		log.Printf("my questions: %v", err)
		return
	}

	totalPages := (res.Total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
		res, err = c.admin.MyQuestions(ctx, userID, page, pageSize)
		if err != nil {
			log.Printf("my questions: %v", err)
			return
		}
	}

	rows := make([][]models.InlineKeyboardButton, 0, len(res.Items)+2)
	for i, q := range res.Items {
		idx := (page-1)*pageSize + i + 1
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         fmt.Sprintf("%d) %s", idx, shortText(q.QuestionText, 35)),
			CallbackData: fmt.Sprintf("adm:open:%d:%d", q.ID, page),
		}})
	}

	nav := []models.InlineKeyboardButton{}
	if page > 1 {
		nav = append(nav, models.InlineKeyboardButton{Text: "⬅️ Пред", CallbackData: fmt.Sprintf("adm:list:%d", page-1)})
	}
	nav = append(nav, models.InlineKeyboardButton{Text: fmt.Sprintf("Страница %d/%d", page, totalPages), CallbackData: "noop"})
	if page < totalPages {
		nav = append(nav, models.InlineKeyboardButton{Text: "➡️ След", CallbackData: fmt.Sprintf("adm:list:%d", page+1)})
	}
	rows = append(rows, nav)
	rows = append(rows, []models.InlineKeyboardButton{{Text: "⬅ Назад", CallbackData: "adm:menu"}})

	text := "Мои вопросы"
	if res.Total == 0 {
		text = "Мои вопросы\n\nПока нет добавленных вопросов"
	}

	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	})
}

func (c *Controller) sendQuestionCard(ctx context.Context, chatID, userID, questionID int64, page int) {
	q, err := c.admin.GetQuestion(ctx, questionID)
	if err != nil || q.Status != schema.QuestionStatusActive {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вопрос больше недоступен"})
		return
	}
	if q.AuthorID != userID {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Это не ваш вопрос"})
		return
	}
	c.sendQuestionCardWithEntity(ctx, chatID, q, page)
}

func (c *Controller) sendQuestionCardWithEntity(ctx context.Context, chatID int64, q schema.Question, page int) {
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text: "Вопрос: " + q.QuestionText,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "👁 Показать ответ", CallbackData: fmt.Sprintf("ans:%d", q.ID)}},
			{{Text: "✏️ Изменить", CallbackData: fmt.Sprintf("adm:edit:%d:%d", q.ID, page)}},
			{{Text: "🗑 Удалить", CallbackData: fmt.Sprintf("adm:delask:%d:%d", q.ID, page)}},
			{{Text: "⬅ Назад к списку", CallbackData: fmt.Sprintf("adm:list:%d", page)}},
		}},
	})
}

func (c *Controller) sendDraftPreview(ctx context.Context, chatID int64, state schema.FormState) {
	buttons := [][]models.InlineKeyboardButton{}
	if state.Mode == schema.FormModeCreate {
		buttons = [][]models.InlineKeyboardButton{
			{{Text: "✅ Подтвердить", CallbackData: "frm:c"}},
			{{Text: "✏️ Изменить", CallbackData: "frm:e"}},
			{{Text: "❌ Отмена", CallbackData: "frm:x"}},
		}
	} else {
		buttons = [][]models.InlineKeyboardButton{
			{{Text: "✅ Сохранить", CallbackData: "frm:s"}},
			{{Text: "✏️ Изменить", CallbackData: "frm:e"}},
			{{Text: "❌ Отмена", CallbackData: "frm:x"}},
		}
	}

	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf("Предпросмотр\n\nВопрос: %s\nОтвет: %s", state.Draft.QuestionText, state.Draft.AnswerText),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
}

func (c *Controller) sendChooseField(ctx context.Context, chatID int64) {
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "Что изменить?",
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Вопрос", CallbackData: "frm:f:q"}},
			{{Text: "Ответ", CallbackData: "frm:f:a"}},
			{{Text: "Назад", CallbackData: "frm:b"}},
		}},
	})
}

func shortText(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

func parseIntPart(data string, idx int) (int, bool) {
	parts := strings.Split(data, ":")
	if len(parts) <= idx {
		return 0, false
	}
	v, err := strconv.Atoi(parts[idx])
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseInt64Part(data string, idx int) (int64, bool) {
	parts := strings.Split(data, ":")
	if len(parts) <= idx {
		return 0, false
	}
	v, err := strconv.ParseInt(parts[idx], 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (c *Controller) answerCallback(ctx context.Context, callbackID, text string) {
	_, _ = c.bot.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       false,
	})
}
