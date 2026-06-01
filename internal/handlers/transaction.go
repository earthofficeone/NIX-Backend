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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionHandler struct {
	tx *repository.TransactionRepository
}

func NewTransactionHandler(tx *repository.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{tx: tx}
}

type txRequest struct {
	Type   string  `json:"type" binding:"required,oneof=income expense"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Title  string  `json:"title"`
	Note   string  `json:"note"`
	Image  string  `json:"image"`
	Date   string  `json:"date" binding:"required"`
}

func (h *TransactionHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	month := c.Query("month")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	list, err := h.tx.ListByUser(ctx, userID, month)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถโหลดรายการได้")
		return
	}

	out := make([]models.TransactionResponse, 0, len(list))
	for _, t := range list {
		out = append(out, t.Response())
	}
	OK(c, out)
}

func (h *TransactionHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสรายการไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tx, err := h.tx.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusNotFound, "ไม่พบรายการ")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถโหลดรายการได้")
		return
	}

	OK(c, tx.Response())
}

func (h *TransactionHandler) Create(c *gin.Context) {
	var req txRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ถูกต้อง")
		return
	}

	userID := middleware.GetUserID(c)
	tx := &models.Transaction{
		UserID: userID,
		Type:   req.Type,
		Amount: req.Amount,
		Title:  defaultTitle(req.Type, req.Title),
		Note:   strings.TrimSpace(req.Note),
		Image:  req.Image,
		Date:   req.Date,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.tx.Create(ctx, tx); err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถบันทึกรายการได้")
		return
	}

	Created(c, tx.Response())
}

func (h *TransactionHandler) Update(c *gin.Context) {
	var req txRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ถูกต้อง")
		return
	}

	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสรายการไม่ถูกต้อง")
		return
	}

	update := bson.M{
		"type":   req.Type,
		"amount": req.Amount,
		"title":  defaultTitle(req.Type, req.Title),
		"note":   strings.TrimSpace(req.Note),
		"date":   req.Date,
	}
	if req.Image != "" {
		update["image"] = req.Image
	} else {
		update["image"] = ""
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tx, err := h.tx.Update(ctx, id, userID, update)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusNotFound, "ไม่พบรายการ")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถแก้ไขรายการได้")
		return
	}

	OK(c, tx.Response())
}

func (h *TransactionHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสรายการไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.tx.Delete(ctx, id, userID); err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถลบรายการได้")
		return
	}

	OK(c, gin.H{"ok": true})
}

func defaultTitle(txType, title string) string {
	t := strings.TrimSpace(title)
	if t != "" {
		return t
	}
	if txType == "income" {
		return "รายรับ"
	}
	return "รายจ่าย"
}
