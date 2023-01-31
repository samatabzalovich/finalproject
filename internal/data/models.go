package data

import (
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

// Define a custom ErrRecordNotFound error. We'll return this from our Get() method when
// looking up a movie that doesn't exist in our database.
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Create a Models struct which wraps the MovieModel. We'll add other models to this,
// like a UserModel and PermissionModel, as our build progresses.
type Models struct {
	Products interface {
		Insert(movie *Product, r *http.Request) error
		Get(id int64, r *http.Request) (*Product, error)
		Update(movie *Product, r *http.Request) error
		Delete(id int64, r *http.Request) error
		GetAll(title string, genres []string, filters Filters, r *http.Request) ([]*Product, Metadata, error)
	}
	Users interface {
		Insert(user *User, r *http.Request) error
		GetByEmail(email string, r *http.Request) (*User, error)
		Update(user *User, r *http.Request) error
		GetForToken(tokenScope, tokenPlaintext string, r *http.Request) (*User, error)
	}
	Tokens interface {
		New(userID int64, ttl time.Duration, scope string) (*Token, error)
		Insert(token *Token) error
		DeleteAllForUser(scope string, userID int64) error
	}
}

// For ease of use, we also add a New() method which returns a Models struct containing
// the initialized MovieModel.
func NewModels(db *pgxpool.Pool) Models {
	m := MovieModel{DB: db}
	u := UserModel{
		DB: db,
	}
	t := TokenModel{
		DB: db,
	}
	return Models{
		Products: m,
		Users:    u,
		Tokens:   t,
	}
}

func NewMockModels() Models {
	return Models{
		Products: MockMovieModel{},
		Users:    MockUserModel{},
		Tokens:   MockTokenModel{},
	}
}
