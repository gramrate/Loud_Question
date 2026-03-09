package telegram

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (c *Controller) mainMenu(userID int64) *models.InlineKeyboardMarkup {
	rows := [][]models.InlineKeyboardButton{
		{{Text: "Играть", CallbackData: "play"}},
		{{Text: "Команда", CallbackData: "team:menu"}},
		{{Text: "Профиль", CallbackData: "profile:menu"}},
	}
	if c.access.IsAdmin(userID) {
		rows = append(rows, []models.InlineKeyboardButton{{Text: "Админка", CallbackData: "adm:menu"}})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (c *Controller) sendProfileMenuWithMessage(ctx context.Context, chatID, userID int64, messageID int) {
	user, ok, err := c.users.GetByID(ctx, userID)
	if err != nil {
		log.Printf("profile get user: %v", err)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось загрузить профиль"})
		return
	}
	if !ok {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Профиль недоступен. Нажмите /start"})
		return
	}

	answeredCnt, err := c.game.AnsweredByUserCount(ctx, userID)
	if err != nil {
		log.Printf("profile answered count: %v", err)
		answeredCnt = 0
	}
	daysSinceReg := int(time.Since(user.RegisteredAt).Hours() / 24)
	if daysSinceReg < 0 {
		daysSinceReg = 0
	}
	daysSinceReg++

	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name == "" {
		name = "Без имени"
	}
	uname := "-"
	if user.Username != "" {
		uname = "@" + user.Username
	}

	text := fmt.Sprintf(
		"Профиль\nИмя: %s\nUsername: %s\nОтветил вопросов: %d\nВ игре уже дней: %d\nID: %d",
		name, uname, answeredCnt, daysSinceReg, userID,
	)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "⬅ Назад", CallbackData: "menu"}},
	}}
	if messageID > 0 {
		_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ReplyMarkup: markup,
		})
		return
	}
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	})
}

func (c *Controller) sendAdminMenu(ctx context.Context, chatID int64) {
	c.sendAdminMenuWithMessage(ctx, chatID, 0)
}

func (c *Controller) sendAdminMenuWithMessage(ctx context.Context, chatID int64, messageID int) {
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "➕ Добавить вопрос", CallbackData: "adm:add"}},
		{{Text: "📥 Добавить Пулл запросов", CallbackData: "adm:pool"}},
		{{Text: "📋 Мои вопросы", CallbackData: "adm:list:1"}},
		{{Text: "⬅ Назад", CallbackData: "menu"}},
	}}
	if messageID > 0 {
		_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        "Админ-панель",
			ReplyMarkup: markup,
		})
		return
	}
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "Админ-панель",
		ReplyMarkup: markup,
	})
}

func (c *Controller) sendPoolPreview(ctx context.Context, chatID int64, state schema.FormState) {
	if state.PoolIndex < 0 || state.PoolIndex >= len(state.PoolItems) {
		return
	}
	item := state.PoolItems[state.PoolIndex]
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf(
			"Пулл вопросов (%d/%d)\n\nВопрос: %s\nОтвет: %s\n\nПодтвердить добавление?",
			state.PoolIndex+1,
			len(state.PoolItems),
			item.QuestionText,
			item.AnswerText,
		),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "✅ Подтвердить", CallbackData: "frm:p:c"}},
			{{Text: "✏️ Изменить", CallbackData: "frm:p:e"}},
			{{Text: "❌ Отмена", CallbackData: "frm:p:x"}},
		}},
	})
}

func (c *Controller) sendMenu(ctx context.Context, chatID, userID int64) {
	c.sendMenuWithMessage(ctx, chatID, userID, 0)
}

func (c *Controller) sendMenuWithMessage(ctx context.Context, chatID, userID int64, messageID int) {
	if messageID > 0 {
		_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        "Главное меню",
			ReplyMarkup: c.mainMenu(userID),
		})
		return
	}
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Главное меню",
		ReplyMarkup: c.mainMenu(userID),
	})
}

func (c *Controller) sendTeamMenu(ctx context.Context, chatID, userID int64) {
	c.sendTeamMenuWithMessage(ctx, chatID, userID, 0)
}

func (c *Controller) sendTeamMenuWithMessage(ctx context.Context, chatID, userID int64, messageID int) {
	team, ok, err := c.team.GetByUserID(ctx, userID)
	if err != nil {
		log.Printf("team by user: %v", err)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ошибка загрузки команды"})
		return
	}
	if !ok {
		markup := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Создать команду", CallbackData: "team:create"}},
			{{Text: "Вступить по UUID", CallbackData: "team:join:help"}},
			{{Text: "⬅ Назад", CallbackData: "menu"}},
		}}
		if messageID > 0 {
			_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
				ChatID:      chatID,
				MessageID:   messageID,
				Text:        "Вы не состоите в команде",
				ReplyMarkup: markup,
			})
			return
		}
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Вы не состоите в команде",
			ReplyMarkup: markup,
		})
		return
	}

	answeredCnt, err := c.game.TeamAnsweredCount(ctx, team.ID)
	if err != nil {
		log.Printf("team answered count: %v", err)
		answeredCnt = 0
	}

	ownerMark := ""
	if team.OwnerID == userID {
		ownerMark = "\nВы создатель команды"
	}
	text := fmt.Sprintf("Команда\nUUID: %s\nОтвечено вопросов: %d%s", team.ID, answeredCnt, ownerMark)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "🔗 Инвайт-ссылка", CallbackData: "team:link"}},
		{{Text: "👥 Участники", CallbackData: "team:members"}},
		{{Text: "🔄 Передать команду", CallbackData: "team:owner:list"}},
		{{Text: "🚪 Выйти из команды", CallbackData: "team:leave"}},
		{{Text: "⬅ Назад", CallbackData: "menu"}},
	}}
	if messageID > 0 {
		_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ReplyMarkup: markup,
		})
		return
	}
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
		ReplyMarkup: markup,
	})
}

func (c *Controller) sendTeamInvite(ctx context.Context, chatID, userID int64) {
	team, ok, err := c.team.GetByUserID(ctx, userID)
	if err != nil {
		log.Printf("team by user: %v", err)
		return
	}
	if !ok {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Сначала вступите в команду"})
		return
	}
	link := fmt.Sprintf("https://t.me/%s?start=jointeam-%s", c.botUsername, team.ID)
	shareText := "Тебя пригласили в команду в Громкий вопрос!"
	shareURL := "https://t.me/share/url?url=" + url.QueryEscape(link) + "&text=" + url.QueryEscape(shareText)
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("Ссылка для входа:\n%s\n\nИли код вручную: %s", link, team.ID),
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📋 Скопировать код", CopyText: models.CopyTextButton{Text: team.ID}}},
			{{Text: "📨 Переслать приглашение", URL: shareURL}},
		}},
	})
}

func (c *Controller) sendTeamMembers(ctx context.Context, chatID, userID int64) {
	c.sendTeamMembersWithMessage(ctx, chatID, userID, 0)
}

func (c *Controller) sendTeamMembersWithMessage(ctx context.Context, chatID, userID int64, messageID int) {
	team, ok, err := c.team.GetByUserID(ctx, userID)
	if err != nil {
		log.Printf("team by user: %v", err)
		return
	}
	if !ok {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы не состоите в команде"})
		return
	}
	members, err := c.team.Members(ctx, team.ID)
	if err != nil {
		log.Printf("team members: %v", err)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось загрузить участников"})
		return
	}

	lines := make([]string, 0, len(members)+1)
	lines = append(lines, "Участники команды:")
	rows := make([][]models.InlineKeyboardButton, 0, len(members)+2)
	for _, m := range members {
		role := "участник"
		if m.UserID == team.OwnerID {
			role = "создатель"
		}
		fullName := strings.TrimSpace(strings.TrimSpace(m.FirstName) + " " + strings.TrimSpace(m.LastName))
		if fullName == "" {
			fullName = "Без имени"
		}
		line := fmt.Sprintf("- %s", fullName)
		if m.Username != "" {
			line += fmt.Sprintf(" | @%s", m.Username)
		}
		line += fmt.Sprintf(" | id=%d (%s)", m.UserID, role)
		lines = append(lines, line)
		if userID == team.OwnerID && m.UserID != team.OwnerID {
			rows = append(rows, []models.InlineKeyboardButton{{
				Text:         fmt.Sprintf("Кикнуть %d", m.UserID),
				CallbackData: fmt.Sprintf("team:kick:%d", m.UserID),
			}})
		}
	}

	rows = append(rows, []models.InlineKeyboardButton{{Text: "⬅ Назад", CallbackData: "team:menu"}})
	text := strings.Join(lines, "\n")
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if messageID > 0 {
		_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ReplyMarkup: markup,
		})
		return
	}
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	})
}

func (c *Controller) sendTeamOwnerTransferMenu(ctx context.Context, chatID, userID int64, messageID int) {
	team, ok, err := c.team.GetByUserID(ctx, userID)
	if err != nil {
		log.Printf("team by user: %v", err)
		return
	}
	if !ok {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Вы не состоите в команде"})
		return
	}
	if team.OwnerID != userID {
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Передавать команду может только создатель"})
		return
	}

	members, err := c.team.Members(ctx, team.ID)
	if err != nil {
		log.Printf("team members: %v", err)
		_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось загрузить участников"})
		return
	}

	rows := make([][]models.InlineKeyboardButton, 0, len(members)+1)
	for _, m := range members {
		if m.UserID == team.OwnerID {
			continue
		}
		fullName := strings.TrimSpace(strings.TrimSpace(m.FirstName) + " " + strings.TrimSpace(m.LastName))
		if fullName == "" {
			fullName = "Без имени"
		}
		label := fullName
		if m.Username != "" {
			label += " | @" + m.Username
		}
		label += fmt.Sprintf(" | id=%d", m.UserID)
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         shortText(label, 60),
			CallbackData: fmt.Sprintf("team:owner:%d", m.UserID),
		}})
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "⬅ Назад", CallbackData: "team:menu"}})

	text := "Кому передать команду?"
	if len(rows) == 1 {
		text = "В команде нет других участников"
	}

	if messageID > 0 {
		_, _ = c.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
		})
		return
	}
	_, _ = c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
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
			CallbackData: fmt.Sprintf("adm:open:%s:%d", q.ID, page),
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

func (c *Controller) sendQuestionCard(ctx context.Context, chatID, userID int64, questionID string, page int) {
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
			{{Text: "👁 Показать ответ", CallbackData: fmt.Sprintf("ans:%s", q.ID)}},
			{{Text: "✏️ Изменить", CallbackData: fmt.Sprintf("adm:edit:%s:%d", q.ID, page)}},
			{{Text: "🗑 Удалить", CallbackData: fmt.Sprintf("adm:delask:%s:%d", q.ID, page)}},
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
