package controllers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/nutrition"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type NutritionController struct {
	profiles *stores.ProfileStore
	foods    *stores.FoodStore
}

func NewNutritionController(profiles *stores.ProfileStore, foods *stores.FoodStore) *NutritionController {
	return &NutritionController{profiles: profiles, foods: foods}
}

func (nc *NutritionController) GetMealPlan(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	profile, err := nc.profiles.GetProfile(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "no profile yet — complete onboarding first")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not load profile")
		return
	}

	targets, err := nc.profiles.GetLatestNutritionTargets(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "no nutrition targets yet — complete onboarding first")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not load nutrition targets")
		return
	}

	foods, err := nc.foods.ListAll()
	if err != nil {
		utils.InternalError(c, "could not load food catalog")
		return
	}

	plan := nutrition.BuildPlan(foods, targets, nutrition.Preferences{
		Likes:        profile.Likes,
		Dislikes:     profile.Dislikes,
		Restrictions: profile.Restrictions,
	})

	utils.OK(c, plan)
}

// GetTargetsHistory powers the "dashboard de ajustes" timeline — every row
// in nutrition_targets is append-only, so this is simply the full history,
// newest first, with no separate audit table needed.
func (nc *NutritionController) GetTargetsHistory(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	history, err := nc.profiles.GetNutritionTargetsHistory(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load your plan history", err)
		return
	}

	utils.OK(c, history)
}

// ListFoods returns the whole curated catalog — the frontend's food-log
// search picker filters it client-side, same approach as the exercise
// picker, since the catalog is small (tens, not thousands, of rows).
func (nc *NutritionController) ListFoods(c *gin.Context) {
	foods, err := nc.foods.ListAll()
	if err != nil {
		utils.InternalErrorf(c, "could not load the food catalog", err)
		return
	}

	utils.OK(c, foods)
}
