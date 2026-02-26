package telegram

import (
	"context"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
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

func (c *Controller) answerCallback(ctx context.Context, callbackID, text string) {
	_, _ = c.bot.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       false,
	})
}
