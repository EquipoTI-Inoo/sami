package controllers

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"coachia-backend/checkin"
	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type CheckinController struct {
	profiles *stores.ProfileStore
}

func NewCheckinController(profiles *stores.ProfileStore) *CheckinController {
	return &CheckinController{profiles: profiles}
}

// GetStatus tells the frontend whether a weekly check-in is due, so the
// Dashboard can show an in-app reminder banner — the PRD decided v0 is
// in-app only, no email, checked on read rather than via a real scheduler.
func (cc *CheckinController) GetStatus(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	latest, err := cc.profiles.GetLatestBodyMeasurement(userID)
	if errors.Is(err, stores.ErrNotFound) {
		// No measurements at all means onboarding hasn't happened yet —
		// nothing to remind about.
		utils.OK(c, models.CheckinStatus{NeedsCheckin: false})
		return
	}
	if err != nil {
		utils.InternalErrorf(c, "could not load check-in status", err)
		return
	}

	due, daysSince, err := checkin.IsDue(latest.LoggedOn, time.Now())
	if err != nil {
		utils.InternalErrorf(c, "could not compute check-in status", err)
		return
	}

	utils.OK(c, models.CheckinStatus{
		NeedsCheckin:  due,
		LastLoggedOn:  latest.LoggedOn,
		DaysSinceLast: daysSince,
	})
}

func (cc *CheckinController) Submit(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	var req models.CheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "missing or invalid check-in fields: "+err.Error())
		return
	}

	var note *string
	if req.Note != "" {
		note = &req.Note
	}

	measurement, err := cc.profiles.InsertBodyMeasurement(userID, stores.BodyMeasurementInput{
		WeightKg:   req.WeightKg,
		WaistCm:    req.WaistCm,
		HipCm:      req.HipCm,
		ArmCm:      req.ArmCm,
		BodyFatPct: req.BodyFatPct,
		Note:       note,
	})
	if err != nil {
		utils.InternalErrorf(c, "could not save your check-in", err)
		return
	}

	utils.Created(c, measurement)
}
