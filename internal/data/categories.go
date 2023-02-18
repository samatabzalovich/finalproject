package data

import (
	"context"
	"database/sql"
	"errors"
	"finalproject/internal/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

type Category struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Image string `json:"image"`
}

func ValidateCategory(v *validator.Validator, category *Category) {
	v.Check(category.Title != "", "title", "must be provided")
	v.Check(len(category.Title) <= 1000, "title", "must not be more than 1000 bytes long")
}

type CategoryModel struct {
	DB *pgxpool.Pool
}

func (m CategoryModel) Insert(category *Category, r *http.Request) error {
	query := `INSERT INTO categories (title, image) VALUES ($1, $2) 
                                                  RETURNING id`
	args := []any{category.Title, category.Image}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRow(ctx, query, args...).Scan(&category.ID)
}
func (m CategoryModel) Get(id int, r *http.Request) (*Category, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `SELECT title, image
				FROM categories
					WHERE id = $1`
	var category Category
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	// Importantly, use defer to make sure that we cancel the context before the Get()
	// method returns.
	defer cancel()

	err := m.DB.QueryRow(ctx, query, id).Scan(
		&category.Title,
		&category.Image,
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
	category.ID = id
	// Otherwise, return a pointer to the Movie struct.
	return &category, nil
}
func (m CategoryModel) GetAll(r *http.Request) ([]*Category, error) {
	query := `SELECT id , title , image
				FROM categories`
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	categories := []*Category{}
	rows, err := m.DB.Query(ctx, query)
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
	defer rows.Close()
	for rows.Next() {
		var category Category
		err := rows.Scan(
			&category.ID,
			&category.Title,
			&category.Image,
		)

		if err != nil {
			return nil, err
		}
		categories = append(categories, &category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

// Add a placeholder method for updating a specific record in the movies table.
func (m CategoryModel) Update(category *Category, r *http.Request) error {
	// Declare the SQL query for updating the record and returning the new version
	// number.
	query := `
		UPDATE categories
			SET title = $1, image = $2
		WHERE id = $3 RETURNING id`
	// Create an args slice containing the values for the placeholder parameters.
	args := []any{
		&category.Title,
		&category.Image,
		&category.ID,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(&category.ID)
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
func (m CategoryModel) Delete(id int, r *http.Request) error {
	// Return an ErrRecordNotFound error if the product ID is less than 1.
	if id < 1 {
		return pgx.ErrNoRows
	}
	// Construct the SQL query to delete the record.
	query := `
		DELETE FROM categories
			WHERE id = $1`
	var category Category
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows := m.DB.QueryRow(ctx, query, id)
	err := rows.Scan(
		&category.ID,
		&category.Title,
		&category.Image,
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
