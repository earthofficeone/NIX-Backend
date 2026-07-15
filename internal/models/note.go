package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NoteBlock struct {
	ID       string `bson:"id" json:"id"`
	Type     string `bson:"type" json:"type"`
	Content  string `bson:"content" json:"content"`
	FileName string `bson:"fileName,omitempty" json:"fileName,omitempty"`
	FileMime string `bson:"fileMime,omitempty" json:"fileMime,omitempty"`
	FileSize int64  `bson:"fileSize,omitempty" json:"fileSize,omitempty"`
}

type Note struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Title     string             `bson:"title" json:"title"`
	Blocks    []NoteBlock        `bson:"blocks" json:"blocks"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type NoteResponse struct {
	ID        string      `json:"id"`
	UserID    string      `json:"userId"`
	Title     string      `json:"title"`
	Blocks    []NoteBlock `json:"blocks"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

func (n Note) Response() NoteResponse {
	blocks := n.Blocks
	if blocks == nil {
		blocks = []NoteBlock{}
	}
	return NoteResponse{
		ID:        n.ID.Hex(),
		UserID:    n.UserID.Hex(),
		Title:     n.Title,
		Blocks:    blocks,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}
