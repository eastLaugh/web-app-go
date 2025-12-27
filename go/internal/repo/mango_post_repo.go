package repo

import (
	"context"

	"time"

	"github.com/eastLaugh/web-app-go/go/internal/api"
	util "github.com/eastLaugh/web-app-go/go/pkg"
	"github.com/oapi-codegen/runtime/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type PostDoc struct {
	Id        bson.ObjectID `bson:"_id,omitempty"`
	File      string        `bson:"file"`
	Content   string        `bson:"content"`
	Email     string        `bson:"email"`
	CreatedAt time.Time     `bson:"created_at"`
}

type MangoPostRepo struct {
	*mongo.Collection
}

func NewMangoPostRepo(collection *mongo.Collection) PostRepo {
	return &MangoPostRepo{collection}
}

func (coll *MangoPostRepo) GetPostsByFile(ctx context.Context, file string) ([]api.Post, error) {
	cursor, err := coll.Find(ctx, bson.M{
		"file": file,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []api.Post
	for cursor.Next(ctx) {
		var post PostDoc
		if err := cursor.Decode(&post); err != nil {
			return nil, err
		}
		posts = append(posts, api.Post{
			Id:        util.New(post.Id.Hex()),
			File:      &post.File,
			Content:   &post.Content,
			Email:     util.New(types.Email(post.Email)),
			CreatedAt: &post.CreatedAt,
		})
	}
	return posts, nil
}

func (coll *MangoPostRepo) InsertPost(ctx context.Context, email string, content string, file string) error {
	_, err := coll.InsertOne(ctx, PostDoc{
		File:      file,
		Content:   content,
		Email:     email,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	return nil
}
