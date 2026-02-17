package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-sad-tg-bot/bot"
	"github.com/go-sad-tg-bot/models"
	"github.com/go-sad-tg-bot/modules/button"
	"github.com/go-sad-tg-bot/modules/wizard"
	"github.com/redis/go-redis/v9"
)

// Структура для хранения данных анкеты пользователя
type UserProfile struct {
	Name      string
	Age       int
	City      string
	Interests []string
}

func main() {
	// 1. Инициализация Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // адрес вашего Redis
		Password: "",               // пароль (если есть)
		DB:       0,                // номер базы
	})

	// Проверяем подключение к Redis
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Не удалось подключиться к Redis: %v", err)
	}
	log.Println("Redis подключен успешно")

	// 2. Создаем хранилище для состояний wizard на базе Redis
	stateStorage := wizard.NewRedisStateStorage(redisClient, "wizard:state:")

	// 3. Создаем конфигурацию бота
	cfg := &bot.Config{
		Token:       os.Getenv("TELEGRAM_BOT_TOKEN"), // Токен из переменной окружения
		WebhookURL:  "",                              // Пустая строка = используем Long Polling
		Debug:       true,                            // Включаем отладку для разработки
		RedisClient: redisClient,                     // Передаем клиент Redis (для других нужд)
	}

	// 4. Создаем экземпляр бота
	b, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	// 5. Регистрируем команды
	registerCommands(b, stateStorage)

	// 6. Запускаем бота
	log.Println("Бот запущен...")
	b.Start()

	// 7. Грациозное завершение
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Завершение работы бота...")
	b.Stop()
}

func registerCommands(b *bot.Bot, stateStorage wizard.StateStorage) {
	// Простая команда /start
	b.RegisterCommand("start", func(ctx context.Context, update *models.Update) {
		b.SendMessage(ctx, update.Message.Chat.ID,
			"Привет! Я демо-бот на goSadTgBot!\n\n"+
				"Доступные команды:\n"+
				"/start - это сообщение\n"+
				"/profile - создать анкету (форма из 4 шагов)\n"+
				"/help - помощь")
	})

	// Команда /help
	b.RegisterCommand("help", func(ctx context.Context, update *models.Update) {
		b.SendMessage(ctx, update.Message.Chat.ID,
			"Я умею:\n"+
				"• Отвечать на команды\n"+
				"• Собирать анкеты через wizard\n"+
				"• Хранить состояния в Redis\n\n"+
				"Попробуйте /profile")
	})

	// Команда /profile - запускает многошаговую форму
	b.RegisterCommand("profile", func(ctx context.Context, update *models.Update) {
		// Создаем новую форму
		profileWizard := wizard.NewWizard(stateStorage, 10*time.Minute)

		// Определяем шаги формы
		steps := []*wizard.Step{
			{
				Name: "name",
				Question: &wizard.Question{
					Text: "Как вас зовут?",
					Type: wizard.QuestionTypeText,
				},
				Validate: func(answer string) error {
					if len(answer) < 2 {
						return fmt.Errorf("имя должно содержать минимум 2 символа")
					}
					return nil
				},
			},
			{
				Name: "age",
				Question: &wizard.Question{
					Text: "Сколько вам лет?",
					Type: wizard.QuestionTypeNumber,
				},
				Validate: func(answer string) error {
					age := 0
					fmt.Sscanf(answer, "%d", &age)
					if age < 1 || age > 150 {
						return fmt.Errorf("возраст должен быть от 1 до 150 лет")
					}
					return nil
				},
			},
			{
				Name: "city",
				Question: &wizard.Question{
					Text: "Из какого вы города?",
					Type: wizard.QuestionTypeText,
				},
			},
			{
				Name: "interests",
				Question: &wizard.Question{
					Text: "Какие у вас интересы? (перечислите через запятую)",
					Type: wizard.QuestionTypeText,
				},
			},
		}

		// Запускаем форму
		profileWizard.Start(ctx, update.Message.Chat.ID, update.Message.From.ID, steps, func(ctx context.Context, userID int64, chatID int64, answers map[string]string) {
			// Это callback, который вызывается после завершения формы

			// Собираем данные в структуру
			profile := &UserProfile{
				Name:      answers["name"],
				City:      answers["city"],
				Interests: parseInterests(answers["interests"]),
			}
			fmt.Sscanf(answers["age"], "%d", &profile.Age)

			// Формируем красивое сообщение с анкетой
			profileText := fmt.Sprintf(
				"✅ Анкета успешно заполнена!\n\n"+
					"📝 Ваши данные:\n"+
					"Имя: %s\n"+
					"Возраст: %d\n"+
					"Город: %s\n"+
					"Интересы: %s",
				profile.Name,
				profile.Age,
				profile.City,
				joinInterests(profile.Interests),
			)

			// Отправляем результат пользователю
			b.SendMessage(ctx, chatID, profileText)

			// Здесь можно сохранить анкету в базу данных
			log.Printf("Получена анкета от пользователя %d: %+v", userID, profile)
		})
	})

	// Обработка колбэков от кнопок (если понадобятся)
	b.RegisterCallbackQuery(func(ctx context.Context, callback *models.CallbackQuery) {
		// Просто отвечаем на нажатие кнопки
		b.AnswerCallbackQuery(ctx, callback.ID, "Вы нажали кнопку!", false)
	})

	// Middleware для логирования всех сообщений
	b.Use(func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, update *models.Update) {
			if update.Message != nil {
				log.Printf("Получено сообщение от @%s: %s",
					update.Message.From.UserName,
					update.Message.Text)
			}
			next(ctx, update)
		}
	})
}

// Вспомогательные функции для работы с интересами
func parseInterests(interestsStr string) []string {
	if interestsStr == "" {
		return []string{}
	}
	// Разбиваем по запятой и убираем лишние пробелы
	parts := strings.Split(interestsStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func joinInterests(interests []string) string {
	if len(interests) == 0 {
		return "не указаны"
	}
	return strings.Join(interests, ", ")
}
