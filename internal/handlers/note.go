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

type NoteHandler struct {
	notes *repository.NoteRepository
}

func NewNoteHandler(notes *repository.NoteRepository) *NoteHandler {
	return &NoteHandler{notes: notes}
}

type noteRequest struct {
	Title  string             `json:"title"`
	Blocks []models.NoteBlock `json:"blocks"`
}

func (h *NoteHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	list, err := h.notes.ListByUser(ctx, userID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถโหลดโน๊ตได้")
		return
	}

	out := make([]models.NoteResponse, 0, len(list))
	for _, n := range list {
		out = append(out, n.Response())
	}
	OK(c, out)
}

func (h *NoteHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสโน๊ตไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	note, err := h.notes.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusNotFound, "ไม่พบโน๊ต")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถโหลดโน๊ตได้")
		return
	}

	OK(c, note.Response())
}

func (h *NoteHandler) Create(c *gin.Context) {
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ถูกต้อง")
		return
	}

	userID := middleware.GetUserID(c)
	note := &models.Note{
		UserID: userID,
		Title:  strings.TrimSpace(req.Title),
		Blocks: normalizeBlocks(req.Blocks),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.notes.Create(ctx, note); err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถสร้างโน๊ตได้")
		return
	}

	Created(c, note.Response())
}

func (h *NoteHandler) Update(c *gin.Context) {
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ข้อมูลไม่ถูกต้อง")
		return
	}

	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสโน๊ตไม่ถูกต้อง")
		return
	}

	update := bson.M{
		"title":  strings.TrimSpace(req.Title),
		"blocks": normalizeBlocks(req.Blocks),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	note, err := h.notes.Update(ctx, id, userID, update)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Fail(c, http.StatusNotFound, "ไม่พบโน๊ต")
			return
		}
		Fail(c, http.StatusInternalServerError, "ไม่สามารถแก้ไขโน๊ตได้")
		return
	}

	OK(c, note.Response())
}

func (h *NoteHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, "รหัสโน๊ตไม่ถูกต้อง")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.notes.Delete(ctx, id, userID); err != nil {
		Fail(c, http.StatusInternalServerError, "ไม่สามารถลบโน๊ตได้")
		return
	}

	OK(c, gin.H{"ok": true})
}

func normalizeBlocks(blocks []models.NoteBlock) []models.NoteBlock {
	if len(blocks) == 0 {
		return []models.NoteBlock{newParagraphBlock("")}
	}

	out := make([]models.NoteBlock, 0, len(blocks))
	for _, b := range blocks {
		blockType := strings.TrimSpace(b.Type)
		if blockType == "" {
			blockType = "paragraph"
		}
		id := strings.TrimSpace(b.ID)
		if id == "" {
			id = primitive.NewObjectID().Hex()
		}
		out = append(out, models.NoteBlock{
			ID:       id,
			Type:     blockType,
			Content:  b.Content,
			FileName: strings.TrimSpace(b.FileName),
			FileMime: strings.TrimSpace(b.FileMime),
			FileSize: b.FileSize,
		})
	}
	return out
}

func newParagraphBlock(content string) models.NoteBlock {
	return models.NoteBlock{
		ID:      primitive.NewObjectID().Hex(),
		Type:    "paragraph",
		Content: content,
	}
}
