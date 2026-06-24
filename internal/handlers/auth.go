package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"nix-backend/internal/config"
	"nix-backend/internal/middleware"
	"nix-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	cfg    config.Config
	users  *repository.UserRepository
	logger *slog.Logger
}

func NewAuthHandler(
	cfg config.Config,
	users *repository.UserRepository,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{cfg: cfg, users: users, logger: logger}
}

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=4"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type resetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required,min=4"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

type messageResponse struct {
	Message string `json:"message"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ครบถ้วน")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	exists, err := h.users.EmailExists(ctx, req.Email)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถสมัครสมาชิกได้")
		return
	}
	if exists {
		Fail(c, http.StatusConflict, "อีเมลนี้ถูกใช้งานแล้ว")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถสมัครสมาชิกได้")
		return
	}

	user, err := h.users.Create(ctx, req.Name, req.Email, string(hash))
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถสมัครสมาชิกได้")
		return
	}

	token, err := middleware.SignToken(user.ID.Hex(), h.cfg.JWTSecret)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถสร้างโทเคนได้")
		return
	}

	Created(c, authResponse{Token: token, User: user.Public()})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "กรุณากรอกอีเมลและรหัสผ่าน")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	user, err := h.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusUnauthorized, "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถเข้าสู่ระบบได้")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		Fail(c, http.StatusUnauthorized, "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		return
	}

	token, err := middleware.SignToken(user.ID.Hex(), h.cfg.JWTSecret)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถสร้างโทเคนได้")
		return
	}

	OK(c, authResponse{Token: token, User: user.Public()})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusUnauthorized, "ไม่พบผู้ใช้")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถโหลดข้อมูลได้")
		return
	}

	OK(c, user.Public())
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ครบถ้วนหรือไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	emailNorm := NormalizeEmail(req.Email)
	code := strings.TrimSpace(req.Code)

	exists, err := h.users.EmailExists(ctx, emailNorm)
	if err != nil {
		h.logger.Error("reset password: check email failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}
	if !exists {
		Fail(c, http.StatusNotFound, "ไม่พบอีเมลนี้ในระบบ")
		return
	}

	if h.cfg.ResetMasterCode == "" || code != h.cfg.ResetMasterCode {
		Fail(c, http.StatusBadRequest, "รหัสลับไม่ถูกต้อง")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถตั้งรหัสผ่านใหม่ได้")
		return
	}

	if err := h.users.UpdatePassword(ctx, emailNorm, string(hash)); err != nil {
		h.logger.Error("reset password: update failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถตั้งรหัสผ่านใหม่ได้")
		return
	}

	OK(c, messageResponse{Message: "ตั้งรหัสผ่านใหม่สำเร็จ"})
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
