package controllers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type FoodLogController struct {
	foodLog  *stores.FoodLogStore
	foods    *stores.FoodStore
	profiles *stores.ProfileStore
}

func NewFoodLogController(foodLog *stores.FoodLogStore, foods *stores.FoodStore, profiles *stores.ProfileStore) *FoodLogController {
	return &FoodLogController{foodLog: foodLog, foods: foods, profiles: profiles}
}

func (fc *FoodLogController) Create(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	var req models.CreateFoodLogEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "food_id, meal_type and servings are required")
		return
	}

	if _, err := fc.foods.GetByID(req.FoodID); errors.Is(err, stores.ErrNotFound) {
		utils.BadRequest(c, "that food doesn't exist")
		return
	} else if err != nil {
		utils.InternalErrorf(c, "could not validate the food", err)
		return
	}

	entry, err := fc.foodLog.Create(userID, req)
	if err != nil {
		utils.InternalErrorf(c, "could not log that food", err)
		return
	}

	utils.Created(c, entry)
}

// GetDay returns everything logged on a date (default today) plus totals
// consumed so far and the current target — everything the Home ring needs
// in one call. Totals are summed from the entries in the same response,
// never stored separately, so they can never drift from what's listed.
func (fc *FoodLogController) GetDay(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	date := c.Query("date")
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	entries, err := fc.foodLog.ListForDay(userID, date)
	if err != nil {
		utils.InternalErrorf(c, "could not load your food log", err)
		return
	}

	var consumed models.MacroTotals
	for _, e := range entries {
		consumed.Calories += e.Calories
		consumed.ProteinG += e.ProteinG
		consumed.CarbsG += e.CarbsG
		consumed.FatG += e.FatG
	}

	var targetPtr *models.NutritionTargets
	if target, err := fc.profiles.GetLatestNutritionTargets(userID); err == nil {
		targetPtr = &target
	} else if !errors.Is(err, stores.ErrNotFound) {
		utils.InternalErrorf(c, "could not load your targets", err)
		return
	}

	utils.OK(c, models.DailyFoodLog{
		LoggedOn: date,
		Entries:  entries,
		Consumed: consumed,
		Target:   targetPtr,
	})
}

func (fc *FoodLogController) Delete(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid entry id")
		return
	}

	owner, err := fc.foodLog.GetOwner(id)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "entry not found")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the entry", err)
		return
	}
	if owner != userID {
		utils.NotFound(c, "entry not found")
		return
	}

	if err := fc.foodLog.Delete(id); err != nil {
		utils.InternalErrorf(c, "could not remove the entry", err)
		return
	}

	utils.OK(c, gin.H{"deleted": true})
}
