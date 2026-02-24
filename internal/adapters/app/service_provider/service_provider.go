package service_provider

import (
	"LoudQuestionBot/internal/adapters/config"
	tgcontroller "LoudQuestionBot/internal/adapters/controller/telegram"
	"LoudQuestionBot/internal/adapters/repository/postgres"
	"LoudQuestionBot/internal/adapters/repository/redisstate"
	"LoudQuestionBot/internal/domain/service/access"
	"LoudQuestionBot/internal/domain/service/admin"
	"LoudQuestionBot/internal/domain/service/form"
	"LoudQuestionBot/internal/domain/service/game"
	telegramsvc "LoudQuestionBot/internal/domain/service/telegram"
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ServiceProvider struct {
	config config.Config

	pgPool      *pgxpool.Pool
	redisClient *redis.Client

	accessService *access.Service
	adminService  *admin.Service
	gameService   *game.Service
	formService   *form.Service

	botRunner telegramsvc.Runner
}

func New() (*ServiceProvider, error) {
	sp := &ServiceProvider{}
	if err := sp.init(); err != nil {
		return nil, err
	}
	return sp, nil
}

func (sp *ServiceProvider) BotRunner() telegramsvc.Runner {
	return sp.botRunner
}

func (sp *ServiceProvider) init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	sp.config = cfg

	ctx := context.Background()

	pgPool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	if err := pgPool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	sp.pgPool = pgPool

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	sp.redisClient = redisClient

	questionRepo := postgres.NewQuestionRepo(sp.pgPool)
	if err := questionRepo.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	formRepo := redisstate.NewFormStateRepo(sp.redisClient)

	sp.accessService = access.New(cfg.AdminIDs)
	sp.adminService = admin.New(questionRepo)
	sp.gameService = game.New(questionRepo)
	sp.formService = form.New(formRepo)

	botRunner, err := tgcontroller.New(cfg.BotToken, sp.accessService, sp.gameService, sp.adminService, sp.formService)
	if err != nil {
		return fmt.Errorf("create telegram controller: %w", err)
	}
	sp.botRunner = botRunner

	log.Println("service provider initialized")
	return nil
}
