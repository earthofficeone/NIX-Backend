package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Type      string             `bson:"type" json:"type"`
	Amount    float64            `bson:"amount" json:"amount"`
	Title     string             `bson:"title" json:"title"`
	Note      string             `bson:"note" json:"note"`
	Image     string             `bson:"image,omitempty" json:"image,omitempty"`
	Date      string             `bson:"date" json:"date"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type TransactionResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"`
	Amount    float64   `json:"amount"`
	Title     string    `json:"title"`
	Note      string    `json:"note"`
	Image     string    `json:"image,omitempty"`
	Date      string    `json:"date"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (t Transaction) Response() TransactionResponse {
	img := t.Image
	return TransactionResponse{
		ID:        t.ID.Hex(),
		UserID:    t.UserID.Hex(),
		Type:      t.Type,
		Amount:    t.Amount,
		Title:     t.Title,
		Note:      t.Note,
		Image:     img,
		Date:      t.Date,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
