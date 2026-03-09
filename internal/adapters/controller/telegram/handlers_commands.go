package telegram

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"errors"
	"fmt"
	"strconv"
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

	registered, isNew, err := c.users.RegisterStart(ctx, schema.BotUser{
		UserID:       userID,
		FirstName:    profile.FirstName,
		LastName:     profile.LastName,
		Username:     profile.Username,
		LanguageCode: strings.TrimSpace(upd.Message.From.LanguageCode),
		IsBot:        upd.Message.From.IsBot,
	})
	if err != nil {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Не удалось зарегистрировать пользователя"})
		return
	}
	_ = c.users.TouchInteraction(ctx, userID)
	if isNew && c.logChatID != 0 {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: c.logChatID,
			Text:   fmt.Sprintf("Новый пользователь:\n%s", formatBotUser(registered)),
		})
	}

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
		ChatID: chatID,
		Text:   "Добро пожаловать в Громкий вопрос",
	})
	c.sendMenu(ctx, chatID, userID)
}

func (c *Controller) menu(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.form.Cancel(ctx, userID)
	_ = c.users.TouchInteraction(ctx, userID)

	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Главное меню",
		ReplyMarkup: c.mainMenu(userID),
	})
}

func (c *Controller) playCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.users.TouchInteraction(ctx, userID)
	c.sendNextQuestion(ctx, chatID, userID)
}

func (c *Controller) teamCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.users.TouchInteraction(ctx, userID)
	c.sendTeamMenu(ctx, chatID, userID)
}

func (c *Controller) profileCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.users.TouchInteraction(ctx, userID)
	c.sendProfileMenuWithMessage(ctx, chatID, userID, 0)
}

func (c *Controller) adminCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.users.TouchInteraction(ctx, userID)
	if !c.access.IsAdmin(userID) {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Недостаточно прав"})
		return
	}
	c.sendAdminMenu(ctx, chatID)
}

func (c *Controller) helpCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.users.TouchInteraction(ctx, userID)

	lines := []string{
		"Доступные команды:",
		"/play - начать игру",
		"/team - меню команды",
		"/profile - ваш профиль",
		"/menu - главное меню",
		"/admin - админ-панель",
		"/help - список команд",
		"/stop - экстренно остановить текущую форму/пулл",
		"/jointeam <uuid> - вступить в команду",
	}
	if c.logChatID != 0 && chatID == c.logChatID {
		lines = append(lines, "/get <id> - информация о пользователе")
	}
	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   strings.Join(lines, "\n"),
	})
}

func (c *Controller) stopCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	userID := upd.Message.From.ID
	_ = c.users.TouchInteraction(ctx, userID)

	state, ok, err := c.form.Get(ctx, userID)
	if err != nil || !ok {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Нет активной операции"})
		return
	}
	added := state.PoolSaved
	total := len(state.PoolItems)
	_ = c.form.Cancel(ctx, userID)
	if state.Step == schema.FormStepPoolInput || state.Step == schema.FormStepPoolPreview || state.Step == schema.FormStepPoolEdit {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("Пулл остановлен. Добавлено: %d из %d", added, total),
		})
		return
	}
	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Операция остановлена"})
}

func (c *Controller) joinTeamByCommand(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	userID := upd.Message.From.ID
	chatID := upd.Message.Chat.ID
	_ = c.users.TouchInteraction(ctx, userID)
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

func (c *Controller) getUserByID(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	_ = c.users.TouchInteraction(ctx, upd.Message.From.ID)
	if c.logChatID == 0 || chatID != c.logChatID {
		return
	}
	args := strings.Fields(strings.TrimSpace(upd.Message.Text))
	if len(args) != 2 {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Использование: /get <id>"})
		return
	}
	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Некорректный id"})
		return
	}
	user, ok, err := c.users.GetByID(ctx, id)
	if err != nil {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Ошибка поиска пользователя"})
		return
	}
	if !ok {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: "Пользователь не нажимал /start или не найден"})
		return
	}
	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   formatBotUser(user),
	})
}

func formatBotUser(user schema.BotUser) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name == "" {
		name = "Без имени"
	}
	uname := "нет"
	if user.Username != "" {
		uname = "@" + user.Username
	}
	return fmt.Sprintf(
		"id: %d\nимя: %s\nusername: %s\nязык: %s\nis_bot: %t\nпоследнее взаимодействие: %s",
		user.UserID,
		name,
		uname,
		valueOrDash(user.LanguageCode),
		user.IsBot,
		user.LastInteractionAt.Format("2006-01-02 15:04:05 MST"),
	)
}

func valueOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}
