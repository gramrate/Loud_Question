package app

import (
	"LoudQuestionBot/internal/adapters/app/service_provider"
	"fmt"
)

func (a *App) initDeps() error {
	inits := []func() error{
		a.initServiceProvider,
	}
	for _, f := range inits {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) initServiceProvider() error {
	sp, err := service_provider.New()
	if err != nil {
		return fmt.Errorf("create service provider: %w", err)
	}
	a.ServiceProvider = sp
	return nil
}
