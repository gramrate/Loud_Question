package main

import (
	"log"

	"LoudQuestionBot/internal/adapters/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}
	a.Start()
}
