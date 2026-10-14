package controllers

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type PendingChangeController struct {
	profiles *stores.ProfileStore
	changes  *stores.PendingChangeStore
}

func NewPendingChangeController(profiles *stores.ProfileStore, changes *stores.PendingChangeStore) *PendingChangeController {
	return &PendingChangeController{profiles: profiles, changes: changes}
}

func (pc *PendingChangeController) ListPending(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	pending, err := pc.changes.ListPending(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load pending changes", err)
		return
	}

	utils.OK(c, pending)
}

func (pc *PendingChangeController) Approve(c *gin.Context) {
	pc.resolve(c, "approved")
}

func (pc *PendingChangeController) Reject(c *gin.Context) {
	pc.resolve(c, "rejected")
}

// resolve is the only place a pending change is ever acted on — approving a
// nutrition_target change writes nutrition_targets right here, same as
// resolve itself; nothing in the chat tool-call path touches that table
// directly (see CoachIA PRD §5.5: "autonomía cero sin confirmación explícita").
func (pc *PendingChangeController) resolve(c *gin.Context, status string) {
	userID := c.GetInt64(middleware.UserIDKey)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "invalid change id")
		return
	}

	change, err := pc.changes.GetByID(id)
	if errors.Is(err, stores.ErrNotFound) {
		utils.NotFound(c, "pending change not found")
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load pending change", err)
		return
	}
	if change.UserID != userID {
		utils.NotFound(c, "pending change not found")
		return
	}
	if change.Status != "pending" {
		utils.Conflict(c, "this change was already resolved")
		return
	}

	if status == "approved" && change.ChangeType == "nutrition_target" {
		var payload models.NutritionTargetChangePayload
		if err := json.Unmarshal(change.Payload, &payload); err != nil {
			utils.InternalErrorf(c, "could not read the proposed change", err)
			return
		}
		if _, err := pc.profiles.InsertNutritionTargets(userID, stores.NutritionTargetsInput{
			Calories:  payload.Calories,
			ProteinG:  payload.ProteinG,
			CarbsG:    payload.CarbsG,
			FatG:      payload.FatG,
			UpdatedBy: "ai",
			Reason:    change.Reason,
		}); err != nil {
			utils.InternalErrorf(c, "could not apply the change", err)
			return
		}
	}

	resolved, err := pc.changes.Resolve(id, status)
	if err != nil {
		utils.InternalErrorf(c, "could not save your decision", err)
		return
	}

	utils.OK(c, resolved)
}
