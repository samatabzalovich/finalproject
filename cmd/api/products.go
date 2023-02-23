package main

import (
	"errors"
	"finalproject/internal/data"
	"finalproject/internal/validator"
	"fmt"
	"net/http"
)

// Add a showMovieHandler for the "GET /v1/movies/:id" endpoint. For now, we retrieve
// the interpolated "id" parameter from the current URL and include it in a placeholder
// response.
func (app *application) showProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Call the Get() method to fetch the data for a specific movie. We also need to
	// use the errors.Is() function to check if it returns a data.ErrRecordNotFound
	// error, in which case we send a 404 Not Found response to the client.
	product, err := app.models.Products.Get(id, r)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"product": product}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string   `json:"title"`
		Owner       int64    `json:"-"`
		Description string   `json:"description"`
		Quantity    int      `json:"quantity"`
		Colors      []string `json:"colors"`
		Images      []string `json:"images"`
		Price       float64  `json:"price"`
		Categories  []int    `json:"categories"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	input.Owner = app.contextGetUser(r).ID
	// Note that the product variable contains a *pointer* to a Movie struct.

	categories := []data.Category{}
	for i := range input.Categories {
		category, err := app.models.Categories.Get(input.Categories[i], r)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.errorResponse(w, r, http.StatusNotFound, "Provided category or categories not found")
			default:
				app.serverErrorResponse(w, r, err)
			}
		}
		categories = append(categories, *category)
	}
	product := &data.Product{
		Title:       input.Title,
		Owner:       input.Owner,
		Description: input.Description,
		Quantity:    input.Quantity,
		Colors:      input.Colors,
		Images:      input.Images,
		Price:       input.Price,
	}
	product.Categories = categories
	v := validator.New()
	if data.ValidateProduct(v, product); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Products.Insert(product, r)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/products/%d", product.ID))
	// Write a JSON response with a 201 Created status code, the product data in the
	// response body, and the Location header.
	err = app.writeJSON(w, http.StatusCreated, envelope{"product": product}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
func (app *application) updateProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Retrieve the movie record as normal.
	product, err := app.models.Products.Get(id, r)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	var input struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Quantity    *int     `json:"quantity"`
		Colors      []string `json:"colors"`
		Images      []string `json:"images"`
		Price       *float64 `json:"price"`
	}
	// Decode the JSON as normal.
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if input.Title != nil {
		product.Title = *input.Title
	}
	// We also do the same for the other fields in the input struct.
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Quantity != nil {
		product.Quantity = *input.Quantity
	}
	if input.Images != nil {
		product.Images = input.Images
	}
	if input.Colors != nil {
		product.Colors = input.Colors // Note that we don't need to dereference a slice.
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	v := validator.New()
	if data.ValidateProduct(v, product); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Products.Update(product, r)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"product": product}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the movie ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Delete the movie from the database, sending a 404 Not Found response to the
	// client if there isn't a matching record.
	err = app.models.Products.Delete(id, r)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return a 200 OK status code along with a success message.
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "product successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
func (app *application) listProductsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title      string
		Categories []string
		data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Title = app.readString(qs, "title", "")
	input.Categories = app.readCSV(qs, "categories", []string{})
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "product_id")
	input.Filters.SortSafelist = []string{"product_id", "title", "price", "quantity", "total_rating", "-product_id", "-title", "-price", "-quantity", "-total_rating"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Accept the metadata struct as a return value.
	products, metadata, err := app.models.Products.GetAll(input.Title, input.Categories, input.Filters, r)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Include the metadata in the response envelope.
	err = app.writeJSON(w, http.StatusOK, envelope{"products": products, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProductId int64  `json:"productId"`
		UserId    int64  `json:"user_id"`
		Rating    int    `json:"rating"`
		Comment   string `json:"comment"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	input.UserId = app.contextGetUser(r).ID
	// Note that the product variable contains a *pointer* to a Movie struct.

	review := &data.RatingSchema{
		UserId:  input.UserId,
		Comment: input.Comment,
		Rating:  input.Rating,
	}
	v := validator.New()
	if data.ValidateReview(v, review); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	ok, err := app.models.Orders.IsUserOrderedProduct(review.UserId, input.ProductId, r)
	if ok != true {
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		app.notPermittedReview(w, r)
		return
	}
	err = app.models.Products.InsertReview(review, input.ProductId, r)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/products/%d", input.ProductId))
	err = app.writeJSON(w, http.StatusCreated, envelope{"review": review}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
