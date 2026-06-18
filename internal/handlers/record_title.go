package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"nix-backend/internal/middleware"
	"nix-backend/internal/models"
	"nix-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecordTitleHandler struct {
	titles *repository.RecordTitleRepository
}

func NewRecordTitleHandler(titles *repository.RecordTitleRepository) *RecordTitleHandler {
	return &RecordTitleHandler{titles: titles}
}

type recordTitleCreateRequest struct {
	Type string `json:"type" binding:"required,oneof=income expense"`
	Name string `json:"name" binding:"required"`
}

type recordTitleUpdateRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *RecordTitleHandler) List(c *gin.Context) {
	txType := c.Query("type")
	if txType != "income" && txType != "expense" {
		Fail(c, http.StatusBadRequest, "ประเภทไม่ถูกต้อง")
		return
	}

	userID := middleware.GetUserID(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	list, err := h.titles.ListByUserAndType(ctx, userID, txType)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถโหลดหัวข้อได้")
		return
	}

	out := make([]models.RecordTitleResponse, 0, len(list))
	for _, item := range list {
		out = append(out, item.Response())
	}
	OK(c, out)
}

func (h *RecordTitleHandler) Create(c *gin.Context) {
	var req recordTitleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ถูกต้อง")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, "กรุณาระบุชื่อหัวข้อ")
		return
	}

	userID := middleware.GetUserID(c)
	title := &models.RecordTitle{
		UserID: userID,
		Type:   req.Type,
		Name:   name,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.titles.Create(ctx, title); err != nil {
		if repository.IsDuplicateKeyError(err) {
			existing, findErr := h.titles.FindByName(ctx, userID, req.Type, name)
			if findErr != nil {
				Fail(c, http.StatusInternalServerError, "ไม่สามารถเพิ่มหัวข้อได้")
				return
			}
			OK(c, existing.Response())
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถเพิ่มหัวข้อได้")
		return
	}

	Created(c, title.Response())
}

func (h *RecordTitleHandler) Update(c *gin.Context) {
	var req recordTitleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ถูกต้อง")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, "กรุณาระบุชื่อหัวข้อ")
		return
	}

	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสหัวข้อไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	current, err := h.titles.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusNotFound, "ไม่พบหัวข้อ")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถแก้ไขหัวข้อได้")
		return
	}

	title, err := h.titles.UpdateName(ctx, id, userID, name)
	if err != nil {
		if repository.IsDuplicateKeyError(err) {
			existing, findErr := h.titles.FindByName(ctx, userID, current.Type, name)
			if findErr != nil {
				Fail(c, http.StatusInternalServerError, "ไม่สามารถแก้ไขหัวข้อได้")
				return
			}
			OK(c, existing.Response())
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถแก้ไขหัวข้อได้")
		return
	}

	OK(c, title.Response())
}

func (h *RecordTitleHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสหัวข้อไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.titles.Delete(ctx, id, userID); err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถลบหัวข้อได้")
		return
	}

	OK(c, gin.H{"ok": true})
}
