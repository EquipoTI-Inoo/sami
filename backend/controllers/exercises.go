package controllers

import (
	"github.com/gin-gonic/gin"

	"coachia-backend/stores"
	"coachia-backend/utils"
)

type ExerciseController struct {
	exercises *stores.ExerciseStore
}

func NewExerciseController(exercises *stores.ExerciseStore) *ExerciseController {
	return &ExerciseController{exercises: exercises}
}

// ListExercises powers the exercise catalog browser — ?location=gym|home
// filters to what's usable there (a "both" exercise always shows), no
// query param returns the whole catalog.
func (ec *ExerciseController) ListExercises(c *gin.Context) {
	location := c.Query("location")
	if location != "" && location != "gym" && location != "home" {
		utils.BadRequest(c, "location must be gym or home")
		return
	}

	exercises, err := ec.exercises.ListAll(location)
	if err != nil {
		utils.InternalErrorf(c, "could not load the exercise catalog", err)
		return
	}

	utils.OK(c, exercises)
}
