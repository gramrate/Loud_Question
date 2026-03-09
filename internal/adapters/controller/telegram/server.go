package telegram

import (
	"LoudQuestionBot/internal/domain/service/access"
	adminsvc "LoudQuestionBot/internal/domain/service/admin"
	"LoudQuestionBot/internal/domain/service/form"
	gamesvc "LoudQuestionBot/internal/domain/service/game"
	teamsvc "LoudQuestionBot/internal/domain/service/team"
	"context"
	"log"

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
	team   *teamsvc.Service

	botUsername string
}

func New(token string, accessSvc *access.Service, gameSvc *gamesvc.Service, adminSvc *adminsvc.Service, formSvc *form.Service, teamSvc *teamsvc.Service) (*Runner, error) {
	ctrl := &Controller{access: accessSvc, game: gameSvc, admin: adminSvc, form: formSvc, team: teamSvc}

	b, err := tgbot.New(token, tgbot.WithDefaultHandler(ctrl.defaultHandler))
	if err != nil {
		return nil, err
	}
	ctrl.bot = b
	me, err := b.GetMe(context.Background())
	if err != nil {
		return nil, err
	}
	ctrl.botUsername = me.Username

	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypePrefix, ctrl.start)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/menu", tgbot.MatchTypeExact, ctrl.menu)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/jointeam", tgbot.MatchTypePrefix, ctrl.joinTeamByCommand)

	return &Runner{bot: b}, nil
}

func (r *Runner) Start(ctx context.Context) {
	log.Println("telegram bot started")
	r.bot.Start(ctx)
}

func (c *Controller) defaultHandler(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	switch {
	case upd.CallbackQuery != nil:
		c.handleCallback(ctx, upd)
	case upd.Message != nil && upd.Message.Text != "":
		c.handleText(ctx, upd)
	}
}
