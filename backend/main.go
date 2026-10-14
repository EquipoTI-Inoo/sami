package main

import (
	"log"

	_ "time/tzdata" // Alpine ships no /usr/share/zoneinfo — without this, LoadLocation silently falls back to UTC in prod only.

	"coachia-backend/ai"
	"coachia-backend/config"
	"coachia-backend/db"
	"coachia-backend/routes"
	"coachia-backend/stores"
)

const claudeModel = "claude-haiku-5-5"

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	if cfg.AnthropicAPIKey == "" {
		log.Fatal("ANTHROPIC_API_KEY is required — the coach chat is core to this app, not optional")
	}

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	users := stores.NewUserStore(conn)
	profiles := stores.NewProfileStore(conn)
	foods := stores.NewFoodStore(conn)
	chats := stores.NewChatStore(conn)
	changes := stores.NewPendingChangeStore(conn)
	exercises := stores.NewExerciseStore(conn)
	workouts := stores.NewWorkoutStore(conn)
	programs := stores.NewProgramStore(conn)
	friendships := stores.NewFriendshipStore(conn)
	sharedGoals := stores.NewSharedGoalStore(conn)
	foodLog := stores.NewFoodLogStore(conn)
	aiClient := ai.NewClient(cfg.AnthropicAPIKey, claudeModel)
	router := routes.Setup(cfg, users, profiles, foods, chats, changes, exercises, workouts, programs, friendships, sharedGoals, foodLog, aiClient)

	log.Printf("sami-backend listening on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
