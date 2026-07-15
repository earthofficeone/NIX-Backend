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

type NoteRepository struct {
	col *mongo.Collection
}

func NewNoteRepository(db *mongo.Database) *NoteRepository {
	return &NoteRepository{col: db.Collection("notes")}
}

func (r *NoteRepository) ListByUser(ctx context.Context, userID primitive.ObjectID) ([]models.Note, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "updatedAt", Value: -1},
	})
	cur, err := r.col.Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []models.Note
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []models.Note{}
	}
	return list, nil
}

func (r *NoteRepository) FindByID(ctx context.Context, id, userID primitive.ObjectID) (*models.Note, error) {
	var note models.Note
	err := r.col.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&note)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) Create(ctx context.Context, note *models.Note) error {
	now := time.Now().UTC()
	note.ID = primitive.NewObjectID()
	note.CreatedAt = now
	note.UpdatedAt = now
	if note.Blocks == nil {
		note.Blocks = []models.NoteBlock{}
	}
	_, err := r.col.InsertOne(ctx, note)
	return err
}

func (r *NoteRepository) Update(ctx context.Context, id, userID primitive.ObjectID, update bson.M) (*models.Note, error) {
	update["updatedAt"] = time.Now().UTC()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var note models.Note
	err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id, "userId": userID}, bson.M{"$set": update}, opts).Decode(&note)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) Delete(ctx context.Context, id, userID primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	return err
}
