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

type TransactionRepository struct {
	col *mongo.Collection
}

func NewTransactionRepository(db *mongo.Database) *TransactionRepository {
	return &TransactionRepository{col: db.Collection("transactions")}
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID primitive.ObjectID, month string) ([]models.Transaction, error) {
	filter := bson.M{"userId": userID}
	if month != "" {
		filter["date"] = bson.M{"$regex": "^" + month}
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "date", Value: -1},
		{Key: "createdAt", Value: -1},
	})
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []models.Transaction
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []models.Transaction{}
	}
	return list, nil
}

func (r *TransactionRepository) FindByID(ctx context.Context, id, userID primitive.ObjectID) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.col.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	now := time.Now().UTC()
	tx.ID = primitive.NewObjectID()
	tx.CreatedAt = now
	tx.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, tx)
	return err
}

func (r *TransactionRepository) Update(ctx context.Context, id, userID primitive.ObjectID, update bson.M) (*models.Transaction, error) {
	update["updatedAt"] = time.Now().UTC()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var tx models.Transaction
	err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id, "userId": userID}, bson.M{"$set": update}, opts).Decode(&tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepository) Delete(ctx context.Context, id, userID primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	return err
}
