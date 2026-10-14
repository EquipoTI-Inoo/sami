package controllers

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/nutrition"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type OnboardingController struct {
	profiles  *stores.ProfileStore
	exercises *stores.ExerciseStore
	programs  *stores.ProgramStore
}

func NewOnboardingController(profiles *stores.ProfileStore, exercises *stores.ExerciseStore, programs *stores.ProgramStore) *OnboardingController {
	return &OnboardingController{profiles: profiles, exercises: exercises, programs: programs}
}

func (oc *OnboardingController) Submit(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	var req models.OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "missing or invalid onboarding fields: "+err.Error())
		return
	}

	profile, targets, err := applyOnboarding(oc.profiles, oc.exercises, oc.programs, userID, req)
	if err != nil {
		utils.InternalError(c, "could not save your profile")
		return
	}

	utils.Created(c, models.OnboardingResponse{Profile: profile, Targets: targets})
}

// applyOnboarding is the single place onboarding data gets written — the
// direct REST endpoint above and the chat tool-call path both go through it,
// so "what onboarding does" never drifts between the two entry points.
func applyOnboarding(profiles *stores.ProfileStore, exercises *stores.ExerciseStore, programs *stores.ProgramStore, userID int64, req models.OnboardingRequest) (models.UserProfile, models.NutritionTargets, error) {
	if err := profiles.UpsertProfile(userID, req); err != nil {
		return models.UserProfile{}, models.NutritionTargets{}, err
	}

	if _, err := profiles.InsertBodyMeasurement(userID, stores.BodyMeasurementInput{WeightKg: req.WeightKg, BodyFatPct: req.BodyFatPct}); err != nil {
		return models.UserProfile{}, models.NutritionTargets{}, err
	}

	calc := nutrition.Calculate(req.Sex, req.Age, req.HeightCm, req.WeightKg, req.ActivityLevel, req.Goal)
	targets, err := profiles.InsertNutritionTargets(userID, stores.NutritionTargetsInput{
		Calories:  calc.Calories,
		ProteinG:  calc.ProteinG,
		CarbsG:    calc.CarbsG,
		FatG:      calc.FatG,
		UpdatedBy: "user",
		Reason:    "onboarding",
	})
	if err != nil {
		return models.UserProfile{}, models.NutritionTargets{}, err
	}

	// Best-effort: the program is a secondary enhancement over the core
	// profile+targets write, so a catalog/program hiccup must never block
	// onboarding itself — it can always be regenerated later from Workouts.
	if _, err := generateProgramForProfile(exercises, programs, userID, req.Goal, req.TrainingLocation); err != nil {
		log.Printf("applyOnboarding: could not generate initial program for user %d: %v", userID, err)
	}

	profile, err := profiles.GetProfile(userID)
	return profile, targets, err
}

func (oc *OnboardingController) GetProfile(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	profile, err := oc.profiles.GetProfile(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "no profile yet — complete onboarding first")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not load profile")
		return
	}

	utils.OK(c, profile)
}

func (oc *OnboardingController) GetNutritionTargets(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	targets, err := oc.profiles.GetLatestNutritionTargets(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "no nutrition targets yet — complete onboarding first")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not load nutrition targets")
		return
	}

	utils.OK(c, targets)
}
