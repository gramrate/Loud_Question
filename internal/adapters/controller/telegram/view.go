package telegram

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"fmt"
	"log"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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
		Text:   "Вопрос: " + q.QuestionText,
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
		Text:   fmt.Sprintf("Предпросмотр\n\nВопрос: %s\nОтвет: %s", state.Draft.QuestionText, state.Draft.AnswerText),
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
