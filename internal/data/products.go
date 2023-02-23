package data

import (
	"context"
	"database/sql"
	"errors"
	"finalproject/internal/validator"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"strconv"
	"time"
)

type RatingSchema struct {
	UserId  int64  `json:"user_id"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}
type Product struct {
	ID           int64           `json:"id"`
	CreatedAt    time.Time       `json:"-"`
	Title        string          `json:"title"`
	Owner        int64           `json:"-"`
	Description  string          `json:"description"`
	Quantity     int             `json:"quantity"`
	Colors       []string        `json:"colors"`
	Images       []string        `json:"images"`
	Price        float64         `json:"price"`
	Version      int             `json:"version"`
	Categories   []Category      `json:"categories"`
	TotalRatings float32         `json:"totalRatings"`
	Ratings      []*RatingSchema `json:"ratings,omitempty"`
}

func ValidateProduct(v *validator.Validator, product *Product) {
	v.Check(product.Title != "", "title", "must be provided")
	v.Check(len(product.Title) <= 1000, "title", "must not be more than 1000 bytes long")
	v.Check(len(product.Description) > 10, "title", "must be more than 10 bytes long")
	v.Check(product.Price != 0, "price", "must be provided")
	v.Check(product.Price > 0, "price", "must be a positive value")
	v.Check(product.Categories != nil, "categories", "must be provided")
	v.Check(product.Owner >= 0, "owner", "must be provided")
	v.Check(len(product.Categories) >= 1, "categories", "must contain at least 1 category")
	v.Check(validator.Unique(product.Categories), "categories", "must not contain duplicate values")
}
func ValidateReview(v *validator.Validator, review *RatingSchema) {
	v.Check(review.Rating >= 0, "rating", "must not be less than 0")
	v.Check(review.UserId >= 0, "user_id", "must be provided")
}

// Define a ProductModel struct type which wraps a sql.DB connection pool.
type ProductModel struct {
	DB *pgxpool.Pool
}

func (m ProductModel) Insert(product *Product, r *http.Request) error {
	query := `INSERT INTO products (title, owner, description, images, colors, quantity, price) VALUES ($1, $2, $3, $4, $5, $6, $7) 
                                                  RETURNING product_id, created_at, version`
	//args := []any{}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRow(ctx, query, product.Title, product.Owner, product.Description, product.Images, product.Colors, product.Quantity, product.Price).Scan(&product.ID, &product.CreatedAt, &product.Version)

	if err != nil {
		return err
	}
	for i := range product.Categories {
		query = `INSERT INTO product_category (product_id, category_id) VALUES ($1, $2::int4)`
		command, err := m.DB.Exec(ctx, query, product.ID, product.Categories[i].ID)
		if err != nil {
			return err
		}
		command.Insert()
	}
	return nil
}
func (m ProductModel) Get(id int64, r *http.Request) (*Product, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	// Define the SQL query for retrieving the product data.
	query := `SELECT product_id, created_at, title, owner, description, images, colors, quantity, price, version
				FROM products
					WHERE product_id = $1`
	var product Product
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	// Importantly, use defer to make sure that we cancel the context before the Get()
	// method returns.
	defer cancel()

	err := m.DB.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.Title,
		&product.Owner,
		&product.Description,
		&product.Images,
		&product.Colors,
		&product.Quantity,
		&product.Price,
		&product.Version,
	)
	// Handle any errors. If there was no matching product found, Scan() will return
	// a sql.ErrNoRows error. We check for this and return our custom ErrRecordNotFound
	// error instead.
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	categories := []Category{}
	query = `SELECT categories.id, categories.title, categories.image FROM categories
                                 join product_category pc on categories.id = pc.category_id
    join products p on pc.product_id = p.product_id WHERE p.product_id = $1`
	tempRows, err := m.DB.Query(ctx, query, product.ID)
	for tempRows.Next() {
		var category Category
		err := tempRows.Scan(
			&category.ID,
			&category.Title,
			&category.Image,
		)
		if err != nil {
			return nil, err // Update this to return an empty Metadata struct.
		}
		categories = append(categories, category)
	}
	if err = tempRows.Err(); err != nil {
		return nil, err // Update this to return an empty Metadata struct.
	}
	product.Categories = categories
	reviews := []*RatingSchema{}
	reviews, err = m.GetReviews(product.ID, r)
	if err != nil {
		return nil, err
	}
	product.Ratings = reviews
	// Otherwise, return a pointer to the Movie struct.
	return &product, nil
}

// Add a placeholder method for updating a specific record in the movies table.
func (m ProductModel) Update(product *Product, r *http.Request) error {
	// Declare the SQL query for updating the record and returning the new version
	// number.
	query := `
		UPDATE products
			SET title = $1, owner = $2, description = $3, images = $4, colors = $5, quantity = $6, price = $7 , version = version + 1
		WHERE product_id = $8 AND version = $9
		RETURNING version`
	// Create an args slice containing the values for the placeholder parameters.
	args := []any{
		&product.Title,
		&product.Owner,
		&product.Description,
		&product.Images,
		&product.Colors,
		&product.Quantity,
		&product.Price,
		&product.ID,
		&product.Version,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(&product.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

// Add a placeholder method for deleting a specific record from the movies table.
func (m ProductModel) Delete(id int64, r *http.Request) error {
	// Return an ErrRecordNotFound error if the product ID is less than 1.
	if id < 1 {
		return pgx.ErrNoRows
	}
	// Construct the SQL query to delete the record.
	query := `
		DELETE FROM products
			WHERE product_id = $1 returning product_id`
	// Execute the SQL query using the Exec() method, passing in the id variable as
	// the value for the placeholder parameter. The Exec() method returns a sql.Result
	// object.
	var product Product
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows := m.DB.QueryRow(ctx, query, id)
	err := rows.Scan(
		&product.ID,
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrRecordNotFound
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}

func (m ProductModel) GetAll(title string, categories []string, filters Filters, r *http.Request) ([]*Product, Metadata, error) {
	var categoriesClause string
	if len(categories) > 1 {
		categoriesClause = "AND pc.category_id IN (%s"
		for i := range categories {
			if i == 0 {
				categoriesClause = fmt.Sprintf(categoriesClause, strconv.Itoa(i+2))
			} else {
				categoriesClause += fmt.Sprintf(", %d", i+2)
			}
		}
		categoriesClause += ")"
	}
	if len(categories) == 1 {
		categoriesClause = "AND pc.category_id IN (%s)"
		categoriesClause = fmt.Sprintf(categoriesClause, categories[0])
	}
	// Construct the SQL query to retrieve all movie records.
	var query = fmt.Sprintf(`
					SELECT count(*) OVER( ),
					       pr.product_id, 
					       pr.created_at, 
					       pr.title, pr.owner,
					       pr.description, pr.images, 
					       pr.colors, pr.quantity, pr.price,
					       pr.version, 
					       COALESCE(r.total_rating, 0) AS total_rating FROM products pr
         			JOIN product_category pc ON pr.product_id = pc.product_id
        			LEFT JOIN (
    					SELECT product_id, avg(rating) AS total_rating
    					FROM ratings
    					GROUP BY product_id
						) r ON pr.product_id = r.product_id 
					WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
					  %s
					GROUP BY pr.product_id, r.total_rating					
					ORDER BY %s %s, pr.product_id ASC
					LIMIT $2 OFFSET $3`, categoriesClause, filters.sortColumn(), filters.sortDirection()) // Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	args := []any{title, filters.limit(), filters.offset()}
	// Use QueryContext() to execute the query. This returns a sql.Rows resultset
	// containing the result.
	rows, err := m.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}

	// Importantly, defer a call to rows.Close() to ensure that the resultset is closed
	// before GetAll() returns.
	defer rows.Close()
	// Initialize an empty slice to hold the movie data.
	// Declare a totalRecords variable.
	totalRecords := 0
	products := []*Product{}
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&totalRecords,
			&product.ID,
			&product.CreatedAt,
			&product.Title,
			&product.Owner,
			&product.Description,
			&product.Images,
			&product.Colors,
			&product.Quantity,
			&product.Price,
			&product.Version,
			&product.TotalRatings,
		)
		if err != nil {
			return nil, Metadata{}, err // Update this to return an empty Metadata struct.
		}
		categories := []Category{}
		query = `SELECT categories.id, categories.title, categories.image FROM categories
                                 join product_category pc on categories.id = pc.category_id
    join products p on pc.product_id = p.product_id WHERE p.product_id = $1`
		tempRows, err := m.DB.Query(ctx, query, product.ID)
		for tempRows.Next() {
			var category Category
			err := tempRows.Scan(
				&category.ID,
				&category.Title,
				&category.Image,
			)
			if err != nil {
				return nil, Metadata{}, err // Update this to return an empty Metadata struct.
			}
			categories = append(categories, category)
		}
		if err = tempRows.Err(); err != nil {
			return nil, Metadata{}, err // Update this to return an empty Metadata struct.
		}
		product.Categories = categories
		reviews := []*RatingSchema{}
		reviews, err = m.GetReviews(product.ID, r)
		if err != nil {
			return nil, Metadata{}, err
		}
		product.Ratings = reviews
		products = append(products, &product)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}
	// Generate a Metadata struct, passing in the total record count and pagination
	// parameters from the client.
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	// Include the metadata struct when returning.
	return products, metadata, nil
}

func (m ProductModel) InsertReview(rating *RatingSchema, productId int64, r *http.Request) error {
	query := `INSERT INTO ratings (product_id, user_id, rating, comment) VALUES ($1, $2, $3,$4) 
                                                  RETURNING product_id`
	//args := []any{}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRow(ctx, query, productId, rating.UserId, rating.Rating, rating.Comment).Scan(&productId)
	if err != nil {
		return err
	}
	return nil
}
func (m ProductModel) GetReviews(productId int64, r *http.Request) ([]*RatingSchema, error) {
	if productId < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT user_id, rating, comment
				FROM ratings
					WHERE product_id = $1`
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.Query(ctx, query, productId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	reviews := []*RatingSchema{}
	for rows.Next() {
		var review RatingSchema
		err := rows.Scan(
			&review.UserId,
			&review.Rating,
			&review.Comment,
		)

		if err != nil {
			return nil, err // Update this to return an empty Metadata struct.
		}
		reviews = append(reviews, &review)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return reviews, nil
}

// Мына астындагы кодка тииспендер
type MockProductModel struct{}

func (m MockProductModel) Insert(movie *Product, r *http.Request) error {
	return nil
}
func (m MockProductModel) Get(id int64, r *http.Request) (*Product, error) {
	// Mock the action...
	return nil, nil
}
func (m MockProductModel) Update(movie *Product, r *http.Request) error {
	// Mock the action...
	return nil
}
func (m MockProductModel) Delete(id int64, r *http.Request) error {
	// Mock the action...
	return nil
}
func (m MockProductModel) GetAll(title string, genres []string, filters Filters, r *http.Request) ([]*Product, Metadata, error) {
	return nil, Metadata{}, nil
}

func (m MockProductModel) GetReviews(productId int64, r *http.Request) ([]*RatingSchema, error) {
	return nil, nil
}
func (m MockProductModel) InsertReview(rating *RatingSchema, productId int64, r *http.Request) error {
	return nil
}
