package repository

import (
	"context"
	"time"

	"nix-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RecordTitleRepository struct {
	col *mongo.Collection
}

func NewRecordTitleRepository(db *mongo.Database) *RecordTitleRepository {
	return &RecordTitleRepository{col: db.Collection("record_titles")}
}

func (r *RecordTitleRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "type", Value: 1},
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func (r *RecordTitleRepository) ListByUserAndType(ctx context.Context, userID primitive.ObjectID, txType string) ([]models.RecordTitle, error) {
	filter := bson.M{"userId": userID, "type": txType}
	opts := options.Find().SetSort(bson.D{
		{Key: "name", Value: 1},
		{Key: "createdAt", Value: 1},
	})
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []models.RecordTitle
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []models.RecordTitle{}
	}
	return list, nil
}

func (r *RecordTitleRepository) FindByID(ctx context.Context, id, userID primitive.ObjectID) (*models.RecordTitle, error) {
	var title models.RecordTitle
	err := r.col.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&title)
	if err != nil {
		return nil, err
	}
	return &title, nil
}

func (r *RecordTitleRepository) FindByName(ctx context.Context, userID primitive.ObjectID, txType, name string) (*models.RecordTitle, error) {
	var title models.RecordTitle
	err := r.col.FindOne(ctx, bson.M{"userId": userID, "type": txType, "name": name}).Decode(&title)
	if err != nil {
		return nil, err
	}
	return &title, nil
}

func (r *RecordTitleRepository) Create(ctx context.Context, title *models.RecordTitle) error {
	now := time.Now().UTC()
	title.ID = primitive.NewObjectID()
	title.CreatedAt = now
	title.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, title)
	return err
}

func (r *RecordTitleRepository) UpdateName(ctx context.Context, id, userID primitive.ObjectID, name string) (*models.RecordTitle, error) {
	update := bson.M{
		"name":      name,
		"updatedAt": time.Now().UTC(),
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var title models.RecordTitle
	err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id, "userId": userID}, bson.M{"$set": update}, opts).Decode(&title)
	if err != nil {
		return nil, err
	}
	return &title, nil
}

func (r *RecordTitleRepository) Delete(ctx context.Context, id, userID primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	return err
}

func IsDuplicateKeyError(err error) bool {
	return err != nil && mongo.IsDuplicateKeyError(err)
}
