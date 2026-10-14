package routes

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"coachia-backend/ai"
	"coachia-backend/config"
	"coachia-backend/controllers"
	"coachia-backend/middleware"
	"coachia-backend/stores"
)

func Setup(
	cfg config.Config,
	users *stores.UserStore,
	profiles *stores.ProfileStore,
	foods *stores.FoodStore,
	chats *stores.ChatStore,
	changes *stores.PendingChangeStore,
	exercises *stores.ExerciseStore,
	workouts *stores.WorkoutStore,
	programs *stores.ProgramStore,
	friendships *stores.FriendshipStore,
	sharedGoals *stores.SharedGoalStore,
	foodLog *stores.FoodLogStore,
	aiClient *ai.Client,
) *gin.Engine {
	r := gin.Default()

	// CORS_ORIGIN is comma-separated: a self-hosted 2-person app is reached from
	// more than one origin at once (e.g. http://localhost:8080 on the PC and
	// http://192.168.1.x:8080 from a phone on the same network).
	origins := strings.Split(cfg.CORSOrigin, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	auth := controllers.NewAuthController(users, cfg.JWTSecret)
	onboarding := controllers.NewOnboardingController(profiles, exercises, programs)
	nutritionCtrl := controllers.NewNutritionController(profiles, foods)
	chatCtrl := controllers.NewChatController(profiles, chats, changes, exercises, programs, aiClient)
	checkinCtrl := controllers.NewCheckinController(profiles)
	pendingChangeCtrl := controllers.NewPendingChangeController(profiles, changes)
	exerciseCtrl := controllers.NewExerciseController(exercises)
	workoutCtrl := controllers.NewWorkoutController(workouts, exercises)
	programCtrl := controllers.NewProgramController(profiles, exercises, programs)
	friendCtrl := controllers.NewFriendController(users, friendships, profiles, workouts)
	sharedGoalCtrl := controllers.NewSharedGoalController(friendships, sharedGoals, users, workouts, profiles)
	foodLogCtrl := controllers.NewFoodLogController(foodLog, foods, profiles)

	v1 := r.Group("/api/v1")
	{
		public := v1.Group("/auth")
		public.POST("/register", auth.Register)
		public.POST("/login", auth.Login)

		protected := v1.Group("")
		protected.Use(middleware.RequireAuth(cfg.JWTSecret))
		protected.GET("/me", auth.Me)
		protected.POST("/onboarding", onboarding.Submit)
		protected.GET("/profile", onboarding.GetProfile)
		protected.GET("/nutrition/targets", onboarding.GetNutritionTargets)
		protected.GET("/nutrition/meal-plan", nutritionCtrl.GetMealPlan)
		protected.GET("/nutrition/targets/history", nutritionCtrl.GetTargetsHistory)
		protected.GET("/foods", nutritionCtrl.ListFoods)
		protected.POST("/food-log", foodLogCtrl.Create)
		protected.GET("/food-log", foodLogCtrl.GetDay)
		protected.DELETE("/food-log/:id", foodLogCtrl.Delete)
		protected.POST("/chat", chatCtrl.SendMessage)
		protected.GET("/chat/history", chatCtrl.GetHistory)
		protected.GET("/checkin/status", checkinCtrl.GetStatus)
		protected.POST("/checkin", checkinCtrl.Submit)
		protected.GET("/pending-changes", pendingChangeCtrl.ListPending)
		protected.POST("/pending-changes/:id/approve", pendingChangeCtrl.Approve)
		protected.POST("/pending-changes/:id/reject", pendingChangeCtrl.Reject)
		protected.GET("/exercises", exerciseCtrl.ListExercises)
		protected.POST("/workouts", workoutCtrl.Start)
		protected.GET("/workouts", workoutCtrl.List)
		protected.GET("/workouts/:id", workoutCtrl.Get)
		protected.POST("/workouts/:id/exercises", workoutCtrl.AddExercise)
		protected.POST("/workouts/:id/exercises/:weId/sets", workoutCtrl.LogSet)
		protected.POST("/workouts/:id/complete", workoutCtrl.Complete)
		protected.GET("/programs/latest", programCtrl.GetLatest)
		protected.POST("/programs/generate", programCtrl.Generate)
		protected.POST("/friends/requests", friendCtrl.SendRequest)
		protected.GET("/friends/requests", friendCtrl.ListIncoming)
		protected.POST("/friends/requests/:id/accept", friendCtrl.Accept)
		protected.POST("/friends/requests/:id/reject", friendCtrl.Reject)
		protected.GET("/friends", friendCtrl.ListFriends)
		protected.GET("/friends/:id/profile", friendCtrl.GetProfile)
		protected.POST("/friends/:id/goals", sharedGoalCtrl.Create)
		protected.GET("/friends/:id/goals", sharedGoalCtrl.List)
		protected.POST("/goals/:goalId/complete", sharedGoalCtrl.Complete)
	}

	return r
}
