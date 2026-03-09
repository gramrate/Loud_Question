package telegram

import (
	"LoudQuestionBot/internal/domain/service/access"
	adminsvc "LoudQuestionBot/internal/domain/service/admin"
	"LoudQuestionBot/internal/domain/service/form"
	gamesvc "LoudQuestionBot/internal/domain/service/game"
	teamsvc "LoudQuestionBot/internal/domain/service/team"
	usersvc "LoudQuestionBot/internal/domain/service/user"
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
	users  *usersvc.Service

	botUsername string
	logChatID   int64
}

func New(token string, logChatID int64, accessSvc *access.Service, gameSvc *gamesvc.Service, adminSvc *adminsvc.Service, formSvc *form.Service, teamSvc *teamsvc.Service, userSvc *usersvc.Service) (*Runner, error) {
	ctrl := &Controller{access: accessSvc, game: gameSvc, admin: adminSvc, form: formSvc, team: teamSvc, users: userSvc, logChatID: logChatID}

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
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/play", tgbot.MatchTypeExact, ctrl.playCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/team", tgbot.MatchTypeExact, ctrl.teamCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/profile", tgbot.MatchTypeExact, ctrl.profileCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/admin", tgbot.MatchTypeExact, ctrl.adminCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/help", tgbot.MatchTypeExact, ctrl.helpCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/jointeam", tgbot.MatchTypePrefix, ctrl.joinTeamByCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/get", tgbot.MatchTypePrefix, ctrl.getUserByID)

	return &Runner{bot: b}, nil
}

func (r *Runner) Start(ctx context.Context) {
	log.Println("telegram bot started")
	r.bot.Start(ctx)
}

func (c *Controller) defaultHandler(ctx context.Context, b *tgbot.Bot, upd *models.Update) {
	c.touchUserInteraction(ctx, upd)
	switch {
	case upd.CallbackQuery != nil:
		c.handleCallback(ctx, upd)
	case upd.Message != nil && upd.Message.Text != "":
		c.handleText(ctx, upd)
	}
}

func (c *Controller) touchUserInteraction(ctx context.Context, upd *models.Update) {
	var userID int64
	switch {
	case upd.CallbackQuery != nil:
		userID = upd.CallbackQuery.From.ID
	case upd.Message != nil && upd.Message.From != nil:
		userID = upd.Message.From.ID
	default:
		return
	}
	if userID == 0 {
		return
	}
	if err := c.users.TouchInteraction(ctx, userID); err != nil {
		log.Printf("touch user interaction: %v", err)
	}
}
