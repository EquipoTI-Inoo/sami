package controllers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type AuthController struct {
	users     *stores.UserStore
	jwtSecret string
}

func NewAuthController(users *stores.UserStore, jwtSecret string) *AuthController {
	return &AuthController{users: users, jwtSecret: jwtSecret}
}

func (ac *AuthController) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "email and password (min 8 chars) are required")
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, "could not process password")
		return
	}

	user, err := ac.users.Create(req.Email, hash)
	if errors.Is(err, stores.ErrEmailTaken) {
		utils.Conflict(c, "email already registered")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not create user")
		return
	}

	token, err := utils.GenerateToken(user.ID, ac.jwtSecret)
	if err != nil {
		utils.InternalError(c, "could not generate token")
		return
	}

	utils.Created(c, models.AuthResponse{Token: token, User: user})
}

func (ac *AuthController) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "email and password are required")
		return
	}

	user, hash, err := ac.users.FindByEmailWithPassword(req.Email)
	if errors.Is(err, stores.ErrNotFound) || !utils.CheckPassword(hash, req.Password) {
		utils.Unauthorized(c, "invalid email or password")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not log in")
		return
	}

	token, err := utils.GenerateToken(user.ID, ac.jwtSecret)
	if err != nil {
		utils.InternalError(c, "could not generate token")
		return
	}

	utils.OK(c, models.AuthResponse{Token: token, User: user})
}

func (ac *AuthController) Me(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	user, err := ac.users.FindByID(userID)
	if errors.Is(err, stores.ErrNotFound) {
		utils.Unauthorized(c, "user no longer exists")
		return
	}
	if err != nil {
		utils.InternalError(c, "could not load user")
		return
	}

	utils.OK(c, user)
}
