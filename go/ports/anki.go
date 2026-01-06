package ports

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	anki_api "github.com/eastLaugh/web-app-go/go/internal/api/anki"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Anki struct {
	*Server
	matches []string
	ankiRepo
}

// GetNextCard implements [anki_api.ServerInterface].
func (a *Anki) GetNextCard(w http.ResponseWriter, r *http.Request) {
	var resp anki_api.GetNextCardResponse

	if len(a.matches) == 0 {
		http.Error(w, "No more cards", http.StatusNoContent)
		return
	}

	resp.File = &a.matches[rand.N(len(a.matches))]

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}

// SubmitReview implements [anki_api.ServerInterface].
func (a *Anki) SubmitReview(w http.ResponseWriter, r *http.Request) {
	// panic("unimplemented")
}

func NewAnki(s *Server, fsys fs.FS, coll *mongo.Collection) anki_api.ServerInterface {
	matches, err := fs.Glob(fsys, "blogs/anki/*.md")
	if err != nil {
		panic(err)
	}

	return &Anki{
		Server:  s,
		matches: matches,
		ankiRepo: mongoAnkiRepo{
			coll: coll,
		},
	}

}

func AnkiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var email string
		if v := r.Context().Value("email"); v == nil {
			http.Error(w, "AnkiMiddleware 未授权", http.StatusUnauthorized)
			return
		} else {
			email = v.(string)
		}

		log.Printf("用户[ %s ] 访问 Anki API", email)
		next.ServeHTTP(w, r)
	})
}

type ankiRepo interface {
	GetLearnerStatus(string) (string, error)
}

type mongoAnkiRepo struct {
	coll *mongo.Collection
}

var errUserNotFound = errors.New("user not found")

type Files map[string]struct {
	File   string    `bson:"file"`
	Expire time.Time `bson:"expire"`
}

type Learner struct {
	Id        bson.ObjectID `bson:"_id,omitempty"`
	Email     string        `bson:"email"`
	CreatedAt time.Time     `bson:"created_at"`
	Duration  time.Duration `bson:"duration"`
	Files     Files         `bson:"files"`
}

func (repo mongoAnkiRepo) GetLearnerStatus(email string) (string, error) {
	return "", nil
}
