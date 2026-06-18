package handlers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"nix-backend/internal/config"
	"nix-backend/internal/email"
	"nix-backend/internal/middleware"
	"nix-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const (
	otpLength         = 6
	otpTTL            = 15 * time.Minute
	otpResendCooldown = 5 * time.Second
	maxOTPAttempts    = 5
)

type AuthHandler struct {
	cfg    config.Config
	users  *repository.UserRepository
	resets *repository.PasswordResetRepository
	mailer email.Sender
	logger *slog.Logger
}

func NewAuthHandler(
	cfg config.Config,
	users *repository.UserRepository,
	resets *repository.PasswordResetRepository,
	mailer email.Sender,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{cfg: cfg, users: users, resets: resets, mailer: mailer, logger: logger}
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

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
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

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "กรุณากรอกอีเมลที่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	emailNorm := NormalizeEmail(req.Email)

	exists, err := h.users.EmailExists(ctx, emailNorm)
	if err != nil {
		h.logger.Error("forgot password: check email failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}
	if !exists {
		Fail(c, http.StatusNotFound, "ไม่พบอีเมลนี้ในระบบ")
		return
	}

	lastAt, err := h.resets.LastCreatedAt(ctx, emailNorm)
	if err != nil {
		h.logger.Error("forgot password: check cooldown failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}
	if lastAt != nil && time.Since(*lastAt) < otpResendCooldown {
		Fail(c, http.StatusTooManyRequests, "กรุณารอสักครู่ก่อนขอรหัสใหม่")
		return
	}

	user, err := h.users.FindByEmail(ctx, emailNorm)
	if err != nil {
		h.logger.Error("forgot password: find user failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}

	code, err := generateOTP(otpLength)
	if err != nil {
		h.logger.Error("forgot password: generate otp failed", "error", err)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("forgot password: hash otp failed", "error", err)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}

	if err := h.resets.DeleteByEmail(ctx, emailNorm); err != nil {
		h.logger.Error("forgot password: delete old otp failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}

	expiresAt := time.Now().UTC().Add(otpTTL)
	if err := h.resets.Create(ctx, emailNorm, string(hash), expiresAt); err != nil {
		h.logger.Error("forgot password: save otp failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}

	if err := h.mailer.Send(ctx, user.Email, "รหัสยืนยันรีเซ็ตรหัสผ่าน NIX", email.PasswordResetBody(code)); err != nil {
		h.logger.Error("forgot password: send email failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถส่งอีเมลได้ กรุณาลองใหม่")
		return
	}

	OK(c, messageResponse{Message: "ส่งรหัสยืนยันไปที่อีเมลแล้ว"})
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

	if h.cfg.ResetMasterCode != "" && code == h.cfg.ResetMasterCode {
		if err := h.resetPasswordWithMasterCode(ctx, c, emailNorm, req.Password); err != nil {
			return
		}
		return
	}

	reset, err := h.resets.FindValid(ctx, emailNorm)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusBadRequest, "รหัสยืนยันไม่ถูกต้องหรือหมดอายุแล้ว")
			return
		}
		h.logger.Error("reset password: find otp failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถดำเนินการได้")
		return
	}

	if reset.Attempts >= maxOTPAttempts {
		_ = h.resets.DeleteByEmail(ctx, emailNorm)
		Fail(c, http.StatusBadRequest, "รหัสยืนยันไม่ถูกต้องหรือหมดอายุแล้ว")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(reset.CodeHash), []byte(code)); err != nil {
		_ = h.resets.IncrementAttempts(ctx, reset.ID)
		Fail(c, http.StatusBadRequest, "รหัสยืนยันไม่ถูกต้อง")
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

	if err := h.resets.DeleteByEmail(ctx, emailNorm); err != nil {
		h.logger.Warn("reset password: cleanup otp failed", "error", err, "email", emailNorm)
	}

	OK(c, messageResponse{Message: "ตั้งรหัสผ่านใหม่สำเร็จ"})
}

func (h *AuthHandler) resetPasswordWithMasterCode(ctx context.Context, c *gin.Context, emailNorm, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถตั้งรหัสผ่านใหม่ได้")
		return err
	}

	if err := h.users.UpdatePassword(ctx, emailNorm, string(hash)); err != nil {
		h.logger.Error("reset password: update failed", "error", err, "email", emailNorm)
		Fail(c, http.StatusInternalServerError, "ไม่สามารถตั้งรหัสผ่านใหม่ได้")
		return err
	}

	_ = h.resets.DeleteByEmail(ctx, emailNorm)
	OK(c, messageResponse{Message: "ตั้งรหัสผ่านใหม่สำเร็จ"})
	return nil
}

func generateOTP(length int) (string, error) {
	var sb strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate otp digit: %w", err)
		}
		sb.WriteByte(byte('0') + byte(n.Int64()))
	}
	return sb.String(), nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
