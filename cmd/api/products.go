package main

import (
	"errors"
	"finalproject/internal/data"
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
	movie, err := app.models.Products.Get(id, r)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
func (app *application) addCategoryHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {

}
func (app *application) updateProductHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) deleteProductHandler(w http.ResponseWriter, r *http.Request) {

}
func (app *application) listProductsHandler(w http.ResponseWriter, r *http.Request) {

}
