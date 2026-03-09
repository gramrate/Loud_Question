package telegram

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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

func parseStringPart(data string, idx int) (string, bool) {
	parts := strings.Split(data, ":")
	if len(parts) <= idx {
		return "", false
	}
	v := strings.TrimSpace(parts[idx])
	if v == "" {
		return "", false
	}
	return v, true
}

func isValidUUID(v string) bool {
	_, err := uuid.Parse(v)
	return err == nil
}

func parseStartJoinTeam(text string) (string, bool) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return "", false
	}
	arg := strings.TrimSpace(parts[1])
	const prefix = "jointeam-"
	if !strings.HasPrefix(arg, prefix) {
		return "", false
	}
	teamID := strings.TrimPrefix(arg, prefix)
	if !isValidUUID(teamID) {
		return "", false
	}
	return teamID, true
}

func userProfileFromTelegramUser(user models.User) schema.UserProfile {
	return schema.UserProfile{
		FirstName: strings.TrimSpace(user.FirstName),
		LastName:  strings.TrimSpace(user.LastName),
		Username:  strings.TrimSpace(user.Username),
	}
}

func truncateForAlert(text string) string {
	const maxLen = 200
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}
	r := []rune(text)
	return string(r[:maxLen-1]) + "…"
}

func (c *Controller) answerCallback(ctx context.Context, callbackID, text string, showAlert bool) {
	_, _ = c.bot.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       showAlert,
	})
}
