package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type User struct {
	Email           string   `bson:"_id"`
	ConversationIDs []string `bson:"conversation_ids"`
}

type MangoUserRepo struct {
	*mongo.Collection
}

func NewMangoUserRepo(coll *mongo.Collection) *MangoUserRepo {
	return &MangoUserRepo{Collection: coll}
}

func (r *MangoUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.FindOne(ctx, bson.M{"_id": email}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *MangoUserRepo) AddConversation(ctx context.Context, email, conversationID string) error {
	_, err := r.UpdateOne(ctx, bson.M{"_id": email},
		bson.D{{Key: "$addToSet", Value: bson.M{"conversation_ids": conversationID}}},
		options.UpdateOne().SetUpsert(true))
	return err
}
