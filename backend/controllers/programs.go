package controllers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/training"
	"coachia-backend/utils"
)

type ProgramController struct {
	profiles  *stores.ProfileStore
	exercises *stores.ExerciseStore
	programs  *stores.ProgramStore
}

func NewProgramController(profiles *stores.ProfileStore, exercises *stores.ExerciseStore, programs *stores.ProgramStore) *ProgramController {
	return &ProgramController{profiles: profiles, exercises: exercises, programs: programs}
}

func (pc *ProgramController) GetLatest(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	program, err := pc.programs.GetLatest(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "no program yet — generate one first")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load your program", err)
		return
	}

	utils.OK(c, program)
}

// Generate lets the person request a fresh program on demand (e.g. after
// changing equipment) without re-running the whole onboarding flow.
func (pc *ProgramController) Generate(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	profile, err := pc.profiles.GetProfile(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "complete onboarding first")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load your profile", err)
		return
	}

	program, err := generateProgramForProfile(pc.exercises, pc.programs, userID, profile.Goal, profile.TrainingLocation)
	if err != nil {
		utils.InternalErrorf(c, "could not generate your program", err)
		return
	}
	if program.ID == 0 {
		utils.InternalError(c, "the exercise catalog has nothing to build a program from")
		return
	}

	utils.Created(c, program)
}

// generateProgramForProfile is the single place a program ever gets built —
// used here and from applyOnboarding (onboarding.go), the same
// single-source-of-truth pattern nutrition targets already follow. Returns
// a zero-value Program (ID 0) with no error when the catalog can't produce
// any day — callers decide whether that's fatal for them.
func generateProgramForProfile(exercises *stores.ExerciseStore, programs *stores.ProgramStore, userID int64, goal, trainingLocation string) (models.Program, error) {
	queryLocation := trainingLocation
	if queryLocation == "both" {
		queryLocation = ""
	}

	catalog, err := exercises.ListAll(queryLocation)
	if err != nil {
		return models.Program{}, err
	}

	plan := training.Generate(catalog, goal)
	if len(plan.Days) == 0 {
		return models.Program{}, nil
	}

	return programs.Create(userID, goal, trainingLocation, plan.Days)
}
