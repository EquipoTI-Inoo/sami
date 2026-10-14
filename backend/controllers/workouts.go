package controllers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type WorkoutController struct {
	workouts  *stores.WorkoutStore
	exercises *stores.ExerciseStore
}

func NewWorkoutController(workouts *stores.WorkoutStore, exercises *stores.ExerciseStore) *WorkoutController {
	return &WorkoutController{workouts: workouts, exercises: exercises}
}

func (wc *WorkoutController) Start(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	workout, err := wc.workouts.Create(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not start the workout", err)
		return
	}

	utils.Created(c, workout)
}

func (wc *WorkoutController) List(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	workouts, err := wc.workouts.ListForUser(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load your workout history", err)
		return
	}

	utils.OK(c, workouts)
}

func (wc *WorkoutController) Get(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	workoutID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid workout id")
		return
	}

	workout, err := wc.workouts.GetByID(workoutID)
	if errors.Is(err, stores.ErrNotFound) || (err == nil && workout.UserID != userID) {
		utils.NotFound(c, "workout not found")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the workout", err)
		return
	}

	utils.OK(c, workout)
}

// AddExercise appends one exercise to an in-progress workout — Gym Mode
// builds the exercise list incrementally as the person picks what's next,
// rather than requiring a full plan up front.
func (wc *WorkoutController) AddExercise(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	workoutID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid workout id")
		return
	}
	if !wc.authorizeWorkout(c, workoutID, userID) {
		return
	}

	var req models.AddWorkoutExerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "exercise_id is required")
		return
	}

	if _, err := wc.exercises.GetByID(req.ExerciseID); errors.Is(err, stores.ErrNotFound) {
		utils.BadRequest(c, "exercise does not exist")
		return
	} else if err != nil {
		utils.InternalErrorf(c, "could not validate the exercise", err)
		return
	}

	workoutExercise, err := wc.workouts.AddExercise(workoutID, req.ExerciseID)
	if err != nil {
		utils.InternalErrorf(c, "could not add the exercise", err)
		return
	}

	utils.Created(c, workoutExercise)
}

func (wc *WorkoutController) LogSet(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	workoutExerciseID, err := strconv.ParseInt(c.Param("weId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid workout exercise id")
		return
	}

	owner, workoutID, err := wc.workouts.GetWorkoutExerciseOwner(workoutExerciseID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "workout exercise not found")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the workout exercise", err)
		return
	}
	if owner != userID || strconv.FormatInt(workoutID, 10) != c.Param("id") {
		utils.NotFound(c, "workout exercise not found")
		return
	}

	var req models.LogSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "reps is required")
		return
	}

	set, err := wc.workouts.LogSet(workoutExerciseID, req.Reps, req.WeightKg)
	if err != nil {
		utils.InternalErrorf(c, "could not log the set", err)
		return
	}

	utils.Created(c, set)
}

func (wc *WorkoutController) Complete(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	workoutID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid workout id")
		return
	}
	if !wc.authorizeWorkout(c, workoutID, userID) {
		return
	}

	workout, err := wc.workouts.Complete(workoutID)
	if err != nil {
		utils.InternalErrorf(c, "could not complete the workout", err)
		return
	}

	utils.OK(c, workout)
}

// authorizeWorkout writes the response itself on failure (404, matching the
// "not found" a non-owner would see) so every handler above can just
// check the bool and return.
func (wc *WorkoutController) authorizeWorkout(c *gin.Context, workoutID, userID int64) bool {
	owner, err := wc.workouts.GetWorkoutOwner(workoutID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "workout not found")
		return false
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the workout", err)
		return false
	}
	if owner != userID {
		utils.NotFound(c, "workout not found")
		return false
	}
	return true
}
