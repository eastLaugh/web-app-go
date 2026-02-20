package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type User struct {
	Email               string            `bson:"_id"`
	ConversationIDs     []string          `bson:"conversation_ids"`
	Conversations       map[string]string `bson:"conversations"`       // convID -> JSON array of messages
	ConversationTitles  map[string]string `bson:"conversation_titles"` // convID -> title
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

func (r *MangoUserRepo) GetConversation(ctx context.Context, email, convID string) (string, error) {
	u, err := r.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return "", err
	}
	if u.Conversations == nil {
		return "", nil
	}
	return u.Conversations[convID], nil
}

func (r *MangoUserRepo) SetConversation(ctx context.Context, email, convID, messagesJSON string) error {
	_, err := r.UpdateOne(ctx, bson.M{"_id": email},
		bson.D{{Key: "$set", Value: bson.M{"conversations." + convID: messagesJSON}}},
		options.UpdateOne().SetUpsert(true))
	return err
}

func (r *MangoUserRepo) SetConversationTitle(ctx context.Context, email, convID, title string) error {
	_, err := r.UpdateOne(ctx, bson.M{"_id": email},
		bson.D{{Key: "$set", Value: bson.M{"conversation_titles." + convID: title}}},
		options.UpdateOne().SetUpsert(true))
	return err
}
