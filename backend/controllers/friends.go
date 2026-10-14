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

type FriendController struct {
	users       *stores.UserStore
	friendships *stores.FriendshipStore
	profiles    *stores.ProfileStore
	workouts    *stores.WorkoutStore
}

func NewFriendController(users *stores.UserStore, friendships *stores.FriendshipStore, profiles *stores.ProfileStore, workouts *stores.WorkoutStore) *FriendController {
	return &FriendController{users: users, friendships: friendships, profiles: profiles, workouts: workouts}
}

// SendRequest looks the target up by email — nothing is visible to either
// side until the recipient explicitly accepts (see CoachIA PRD decision:
// "solicitud por email + aceptar", never automatic visibility).
func (fc *FriendController) SendRequest(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "a valid email is required")
		return
	}

	target, err := fc.users.FindByEmail(req.Email)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "no account with that email")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not look up that email", err)
		return
	}
	if target.ID == userID {
		utils.BadRequest(c, "you can't send a friend request to yourself")
		return
	}

	existing, err := fc.friendships.FindBetween(userID, target.ID)
	if err != nil && !errors.Is(err, stores.ErrNotFound) {
		utils.InternalErrorf(c, "could not check existing friendships", err)
		return
	}
	if err == nil && (existing.Status == "accepted" || existing.Status == "pending") {
		utils.Conflict(c, "you're already friends, or a request between you two is already pending")
		return
	}
	// A past 'rejected' friendship falls through — a fresh request is allowed.

	friendship, err := fc.friendships.Create(userID, target.ID)
	if err != nil {
		utils.InternalErrorf(c, "could not send the friend request", err)
		return
	}

	utils.Created(c, friendship)
}

func (fc *FriendController) ListIncoming(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	requests, err := fc.friendships.ListIncomingPending(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load incoming requests", err)
		return
	}

	utils.OK(c, requests)
}

func (fc *FriendController) ListFriends(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	friends, err := fc.friendships.ListAccepted(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load your friends", err)
		return
	}

	utils.OK(c, friends)
}

func (fc *FriendController) Accept(c *gin.Context) {
	fc.resolveIncoming(c, "accepted")
}

func (fc *FriendController) Reject(c *gin.Context) {
	fc.resolveIncoming(c, "rejected")
}

// resolveIncoming only lets the recipient decide — the requester sent it
// and just waits, same as pending_changes only the proposal's subject can
// approve or reject.
func (fc *FriendController) resolveIncoming(c *gin.Context, status string) {
	userID := c.GetInt64(middleware.UserIDKey)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid request id")
		return
	}

	friendship, err := fc.friendships.GetByID(id)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "friend request not found")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the friend request", err)
		return
	}
	if friendship.RecipientID != userID {
		utils.NotFound(c, "friend request not found")
		return
	}
	if friendship.Status != "pending" {
		utils.Conflict(c, "this request was already resolved")
		return
	}

	resolved, err := fc.friendships.Resolve(id, status)
	if err != nil {
		utils.InternalErrorf(c, "could not save your decision", err)
		return
	}

	utils.OK(c, resolved)
}

// GetProfile returns a friend's shared profile — current plan/goal, recent
// check-ins and workout counts — gated by authorizeFriendship below.
func (fc *FriendController) GetProfile(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	friendshipID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid friendship id")
		return
	}

	_, friendUserID, ok := authorizeFriendship(c, fc.friendships, friendshipID, userID)
	if !ok {
		return
	}

	friendUser, err := fc.users.FindByID(friendUserID)
	if err != nil {
		utils.InternalErrorf(c, "could not load your friend's account", err)
		return
	}

	profile, err := fc.profiles.GetProfile(friendUserID)
	var profilePtr *models.UserProfile
	if err == nil {
		profilePtr = &profile
	} else if !errors.Is(err, stores.ErrNotFound) {
		utils.InternalErrorf(c, "could not load your friend's profile", err)
		return
	}

	targets, err := fc.profiles.GetLatestNutritionTargets(friendUserID)
	var targetsPtr *models.NutritionTargets
	if err == nil {
		targetsPtr = &targets
	} else if !errors.Is(err, stores.ErrNotFound) {
		utils.InternalErrorf(c, "could not load your friend's targets", err)
		return
	}

	checkins, err := fc.profiles.ListRecentBodyMeasurements(friendUserID, 8)
	if err != nil {
		utils.InternalErrorf(c, "could not load your friend's check-ins", err)
		return
	}

	workoutsThisWeek, err := fc.workouts.CountCompletedSince(friendUserID, startOfWeek(time.Now()))
	if err != nil {
		utils.InternalErrorf(c, "could not load your friend's workouts", err)
		return
	}
	totalWorkouts, err := fc.workouts.CountCompleted(friendUserID)
	if err != nil {
		utils.InternalErrorf(c, "could not load your friend's workouts", err)
		return
	}

	utils.OK(c, models.FriendProfile{
		Email:            friendUser.Email,
		Profile:          profilePtr,
		Targets:          targetsPtr,
		RecentCheckins:   checkins,
		WorkoutsThisWeek: workoutsThisWeek,
		TotalWorkouts:    totalWorkouts,
	})
}

// authorizeFriendship resolves friendshipID to the other participant's user
// id, but only when it's accepted and userID is actually one of its two
// sides — writing the 404 response itself on any failure, same "not found"
// a non-participant would see, so this never confirms a friendship id's
// existence to someone outside it. Shared by FriendController and
// SharedGoalController.
func authorizeFriendship(c *gin.Context, friendships *stores.FriendshipStore, friendshipID, userID int64) (models.Friendship, int64, bool) {
	friendship, err := friendships.GetByID(friendshipID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "friendship not found")
		return models.Friendship{}, 0, false
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load the friendship", err)
		return models.Friendship{}, 0, false
	}
	if friendship.Status != "accepted" {
		utils.NotFound(c, "friendship not found")
		return models.Friendship{}, 0, false
	}

	switch userID {
	case friendship.RequesterID:
		return friendship, friendship.RecipientID, true
	case friendship.RecipientID:
		return friendship, friendship.RequesterID, true
	default:
		utils.NotFound(c, "friendship not found")
		return models.Friendship{}, 0, false
	}
}

// startOfWeek returns the UTC timestamp for the most recent Monday
// (ISO week start), as a string comparable against the TEXT timestamp
// columns used throughout the schema.
func startOfWeek(t time.Time) string {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // ISO: Monday=1 .. Sunday=7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05.000Z")
}
