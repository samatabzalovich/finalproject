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
	ErrOutOfStock     = errors.New("out of stock")
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
		GetReviews(productId int64, r *http.Request) ([]*RatingSchema, error)
		InsertReview(rating *RatingSchema, productId int64, r *http.Request) error
	}
	Orders interface {
		Insert(userId int64, order *Order, r *http.Request) error
		Get(id int64, r *http.Request) (*Order, error)
		GetAllOrdersForUser(userId int64, filters Filters, r *http.Request) ([]*Order, Metadata, error)
		IsUserOrderedProduct(userId int64, productId int64, r *http.Request) (bool, error)
		Update(order *Order, r *http.Request) error
		Delete(id int64, r *http.Request) error
	}
	Categories interface {
		Insert(category *Category, r *http.Request) error
		Get(id int, r *http.Request) (*Category, error)
		Update(movie *Category, r *http.Request) error
		Delete(id int, r *http.Request) error
		GetAll(r *http.Request) ([]*Category, error)
	}
	Users interface {
		Insert(user *User, r *http.Request) error
		GetByEmail(email string, r *http.Request) (*User, error)
		Update(user *User, r *http.Request) error
		GetForToken(tokenScope, tokenPlaintext string, r *http.Request) (*User, error)
	}
	Permissions interface {
		GetAllForUser(userID int64) (Permissions, error)
		InsertPermissionRead(userID int64) error
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
	m := ProductModel{DB: db}
	u := UserModel{
		DB: db,
	}
	t := TokenModel{
		DB: db,
	}
	c := CategoryModel{
		DB: db,
	}
	return Models{
		Products:   m,
		Users:      u,
		Tokens:     t,
		Categories: c,
		Permissions: PermissionModel{
			db,
		},
		Orders: OrderModel{
			DB: db,
		},
	}
}

func NewMockModels() Models {
	return Models{
		Products: MockProductModel{},
		Users:    MockUserModel{},
		Tokens:   MockTokenModel{},
	}
}
