package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecordTitle struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Type      string             `bson:"type" json:"type"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type RecordTitleResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (t RecordTitle) Response() RecordTitleResponse {
	return RecordTitleResponse{
		ID:        t.ID.Hex(),
		UserID:    t.UserID.Hex(),
		Type:      t.Type,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
