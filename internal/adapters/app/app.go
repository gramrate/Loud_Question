package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"LoudQuestionBot/internal/adapters/app/service_provider"
)

type App struct {
	ServiceProvider *service_provider.ServiceProvider
}

func New() (*App, error) {
	a := &App{}
	if err := a.initDeps(); err != nil {
		return nil, fmt.Errorf("init deps: %w", err)
	}
	return a, nil
}

func (a *App) Start() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	a.ServiceProvider.BotRunner().Start(ctx)
}
