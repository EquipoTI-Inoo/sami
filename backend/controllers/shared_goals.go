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

type SharedGoalController struct {
	friendships *stores.FriendshipStore
	goals       *stores.SharedGoalStore
	users       *stores.UserStore
	workouts    *stores.WorkoutStore
	profiles    *stores.ProfileStore
}

func NewSharedGoalController(friendships *stores.FriendshipStore, goals *stores.SharedGoalStore, users *stores.UserStore, workouts *stores.WorkoutStore, profiles *stores.ProfileStore) *SharedGoalController {
	return &SharedGoalController{friendships: friendships, goals: goals, users: users, workouts: workouts, profiles: profiles}
}

func (gc *SharedGoalController) Create(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	friendshipID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid friendship id")
		return
	}
	if _, _, ok := authorizeFriendship(c, gc.friendships, friendshipID, userID); !ok {
		return
	}

	var req models.CreateSharedGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "goal_type and description are required")
		return
	}

	goal, err := gc.goals.Create(friendshipID, userID, req)
	if err != nil {
		utils.InternalErrorf(c, "could not create the shared goal", err)
		return
	}

	utils.Created(c, goal)
}

// List returns every shared goal for the friendship with each
// participant's progress computed live — see computeProgress. Nothing
// about progress is ever stored, so it can never drift from the real data.
func (gc *SharedGoalController) List(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	friendshipID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid friendship id")
		return
	}
	friendship, _, ok := authorizeFriendship(c, gc.friendships, friendshipID, userID)
	if !ok {
		return
	}

	goals, err := gc.goals.ListForFriendship(friendshipID)
	if err != nil {
		utils.InternalErrorf(c, "could not load shared goals", err)
		return
	}

	for i := range goals {
		progress, err := gc.computeProgress(goals[i], friendship)
		if err != nil {
			utils.InternalErrorf(c, "could not compute goal progress", err)
			return
		}
		goals[i].Progress = progress
	}

	utils.OK(c, goals)
}

// Complete lets either participant mark a shared goal done — it belongs to
// the friendship, not to whoever created it.
func (gc *SharedGoalController) Complete(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	id, err := strconv.ParseInt(c.Param("goalId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid goal id")
		return
	}

	goal, err := gc.goals.GetByID(id)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "goal not found")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the goal", err)
		return
	}

	if _, _, ok := authorizeFriendship(c, gc.friendships, goal.FriendshipID, userID); !ok {
		return
	}

	completed, err := gc.goals.Complete(id)
	if err != nil {
		utils.InternalErrorf(c, "could not complete the goal", err)
		return
	}

	utils.OK(c, completed)
}

// computeProgress reports each side of the friendship's own relevant metric
// for the goal's type — a shared goal's "progress" is read, never a stored
// number, so it's always consistent with the real workouts/check-ins.
func (gc *SharedGoalController) computeProgress(goal models.SharedGoal, friendship models.Friendship) (map[string]models.GoalProgress, error) {
	progress := map[string]models.GoalProgress{}

	for _, userID := range []int64{friendship.RequesterID, friendship.RecipientID} {
		user, err := gc.users.FindByID(userID)
		if err != nil {
			return nil, err
		}
		p := models.GoalProgress{UserID: userID, Email: user.Email}

		switch goal.GoalType {
		case "workouts_per_week":
			count, err := gc.workouts.CountCompletedSince(userID, startOfWeek(time.Now()))
			if err != nil {
				return nil, err
			}
			p.WorkoutsThisWeek = count
		case "weight_target":
			latest, err := gc.profiles.GetLatestBodyMeasurement(userID)
			if err == nil {
				w := latest.WeightKg
				p.CurrentWeightKg = &w
			} else if !errors.Is(err, stores.ErrNotFound) {
				return nil, err
			}
		}

		progress[strconv.FormatInt(userID, 10)] = p
	}

	return progress, nil
}
