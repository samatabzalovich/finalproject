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
	"time"
)

type RatingSchema struct {
	UserId string `json:"user_id"`
	Rating int    `json:"rating"`
}
type Product struct {
	ID          int64          `json:"id"`
	CreatedAt   time.Time      `json:"-"`
	Title       string         `json:"title"`
	Owner       int64          `json:"owner"`
	Description string         `json:"description"`
	Runtime     Runtime        `json:"runtime,omitempty"`
	Categories  []string       `json:"categories"`
	Ratings     []RatingSchema `json:"ratings,omitempty"`
	Version     string         `json:"version"`
}

func ValidateMovie(v *validator.Validator, product *Product) {
	v.Check(product.Title != "", "title", "must be provided")
	v.Check(len(product.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(product.Runtime != 0, "runtime", "must be provided")
	v.Check(product.Runtime > 0, "runtime", "must be a positive integer")
	v.Check(product.Categories != nil, "genres", "must be provided")
	v.Check(product.Owner >= 0, "owner", "must be provided")
	v.Check(len(product.Categories) >= 1, "genres", "must contain at least 1 category")
	v.Check(len(product.Categories) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(product.Categories), "genres", "must not contain duplicate values")
}

// Define a MovieModel struct type which wraps a sql.DB connection pool.
type MovieModel struct {
	DB *pgxpool.Pool
}

func (m MovieModel) Insert(movie *Product, r *http.Request) error {
	return nil
}

func (m MovieModel) Get(id int64, r *http.Request) (*Product, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	// Define the SQL query for retrieving the movie data.
	query := `SELECT id, created_at, title, year, runtime, genres, version
				FROM products
					WHERE id = $1`
	// Declare a Movie struct to hold the data returned by the query.
	var movie Product
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		//&movie.Year,
		&movie.Runtime,
		//&movie.Genres,
		&movie.Version,
	)
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
	// Otherwise, return a pointer to the Movie struct.
	return &movie, nil
}

// Add a placeholder method for updating a specific record in the movies table.
func (m MovieModel) Update(movie *Product, r *http.Request) error {
	// Declare the SQL query for updating the record and returning the new version
	// number.
	query := `
		UPDATE products
			SET title = $1, year = $2, runtime = $3, genres = $4, version = uuid_generate_v4()
		WHERE id = $5 AND version = $6
		RETURNING version`
	// Create an args slice containing the values for the placeholder parameters.
	args := []any{
		movie.Title,
		//movie.Year,
		movie.Runtime,
		//movie.Genres,
		movie.ID,
		movie.Version,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(&movie.Version)
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
func (m MovieModel) Delete(id int64, r *http.Request) error {
	// Return an ErrRecordNotFound error if the movie ID is less than 1.
	if id < 1 {
		return pgx.ErrNoRows
	}
	// Construct the SQL query to delete the record.
	query := `
		DELETE FROM Product
			WHERE id = $1`
	// Execute the SQL query using the Exec() method, passing in the id variable as
	// the value for the placeholder parameter. The Exec() method returns a sql.Result
	// object.
	var movie Product
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows := m.DB.QueryRow(ctx, query, id)
	err := rows.Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		//&movie.Year,
		&movie.Runtime,
		//&movie.Genres,
		&movie.Version)
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
	// If no rows were affected, we know that the movies table didn't contain a record
	// with the provided ID at the moment we tried to delete it. In that case we
	// return an ErrRecordNotFound error.

	return nil
}

// Create a new GetAll() method which returns a slice of movies. Although we're not
// using them right now, we've set this up to accept the various filter parameters as
// arguments.
func (m MovieModel) GetAll(title string, genres []string, filters Filters, r *http.Request) ([]*Product, Metadata, error) {
	// Construct the SQL query to retrieve all movie records.
	query := fmt.Sprintf(`
					SELECT count(*) OVER(), id, created_at, title, year, runtime, genres, version
					FROM products
					WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
					AND (genres @> $2 OR $2 = '{}')
					ORDER BY %s %s, id ASC
					LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	args := []any{title, genres, filters.limit(), filters.offset()}
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
	movies := []*Product{}
	for rows.Next() {
		var movie Product
		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			//&movie.Year,
			&movie.Runtime,
			//&movie.Genres,
			&movie.Version,
		)

		if err != nil {
			return nil, Metadata{}, err // Update this to return an empty Metadata struct.
		}
		movies = append(movies, &movie)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}
	// Generate a Metadata struct, passing in the total record count and pagination
	// parameters from the client.
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	// Include the metadata struct when returning.
	return movies, metadata, nil
}

// Мына астындагы кодка тииспендер
type MockMovieModel struct{}

func (m MockMovieModel) Insert(movie *Product, r *http.Request) error {
	return nil
}
func (m MockMovieModel) Get(id int64, r *http.Request) (*Product, error) {
	// Mock the action...
	return nil, nil
}
func (m MockMovieModel) Update(movie *Product, r *http.Request) error {
	// Mock the action...
	return nil
}
func (m MockMovieModel) Delete(id int64, r *http.Request) error {
	// Mock the action...
	return nil
}
func (m MockMovieModel) GetAll(title string, genres []string, filters Filters, r *http.Request) ([]*Product, Metadata, error) {
	return nil, Metadata{}, nil
}
