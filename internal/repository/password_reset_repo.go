package repository

import (
	"context"
	"strings"
	"time"

	"nix-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PasswordResetRepository struct {
	col *mongo.Collection
}

func NewPasswordResetRepository(db *mongo.Database) *PasswordResetRepository {
	return &PasswordResetRepository{col: db.Collection("password_resets")}
}

func (r *PasswordResetRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

func (r *PasswordResetRepository) DeleteByEmail(ctx context.Context, email string) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"email": normalizeEmail(email)})
	return err
}

func (r *PasswordResetRepository) Create(ctx context.Context, email, codeHash string, expiresAt time.Time) error {
	now := time.Now().UTC()
	_, err := r.col.InsertOne(ctx, models.PasswordReset{
		Email:     normalizeEmail(email),
		CodeHash:  codeHash,
		Attempts:  0,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	})
	return err
}

func (r *PasswordResetRepository) FindValid(ctx context.Context, email string) (*models.PasswordReset, error) {
	var reset models.PasswordReset
	err := r.col.FindOne(ctx, bson.M{
		"email":     normalizeEmail(email),
		"expiresAt": bson.M{"$gt": time.Now().UTC()},
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&reset)
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

func (r *PasswordResetRepository) IncrementAttempts(ctx context.Context, id interface{}) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"attempts": 1}})
	return err
}

func (r *PasswordResetRepository) LastCreatedAt(ctx context.Context, email string) (*time.Time, error) {
	var reset models.PasswordReset
	err := r.col.FindOne(ctx, bson.M{"email": normalizeEmail(email)},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	).Decode(&reset)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &reset.CreatedAt, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
