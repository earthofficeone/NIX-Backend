package repository

import (
	"context"
	"strings"
	"time"

	"nix-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

func (r *UserRepository) Create(ctx context.Context, name, email, hashedPassword string) (*models.User, error) {
	now := time.Now().UTC()
	user := models.User{
		ID:        primitive.NewObjectID(),
		Name:      strings.TrimSpace(name),
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Password:  hashedPassword,
		CreatedAt: now,
	}
	_, err := r.col.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.col.FindOne(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email))}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	count, err := r.col.CountDocuments(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email))})
	return count > 0, err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, email, hashedPassword string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"email": strings.ToLower(strings.TrimSpace(email))},
		bson.M{"$set": bson.M{"password": hashedPassword}},
	)
	return err
}
