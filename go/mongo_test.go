package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/eastLaugh/web-app-go/go/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var mg *mongo.Client
var anki *mongo.Collection

func init() {
	mg = initMongo()
	anki = mg.Database("webapp").Collection("anki")
}

func TestBSON(t *testing.T) {
	lner := ports.Learner{
		Email:     "test@test.com",
		Id:        bson.ObjectID([12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xDC, 0xAA}),
		CreatedAt: time.Now(),
		Files: ports.Files{
			"test.md": {
				File:   "test.md",
				Expire: time.Now().Add(10 * time.Second),
			},
		},
	}

	r, err := anki.InsertOne(context.TODO(), lner)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+#v", r)
}

func TestFind(t *testing.T) {
	res := anki.FindOne(context.TODO(), bson.M{})
	var lner ports.Learner
	res.Decode(&lner)

}

func TestJson(t *testing.T) {
	type User struct {
		Name string // 没有 tag
		Age  int    `json:"age"` // 有 tag，指定为小写
	}

	u := User{Name: "张三", Age: 25}
	data, _ := json.Marshal(u)
	fmt.Println(string(data))
}
