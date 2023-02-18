package main

import (
	"errors"
	"finalproject/internal/data"
	"finalproject/internal/validator"
	"fmt"
	"net/http"
)

func (app *application) orderProductHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID     int64            `json:"-"`
		OrderItems []data.OrderItem `json:"orderItems"`
		Address    string           `json:"address"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	input.UserID = app.contextGetUser(r).ID
	// Note that the product variable contains a *pointer* to a Movie struct.
	order := &data.Order{
		UserId:     input.UserID,
		Address:    input.Address,
		OrderItems: input.OrderItems,
	}
	v := validator.New()
	if data.ValidateOrder(v, order); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Orders.Insert(input.UserID, order, r)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/orders/%d", order.ID))
	// Write a JSON response with a 201 Created status code, the product data in the
	// response body, and the Location header.
	err = app.writeJSON(w, http.StatusCreated, envelope{"order": order}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listUserOrdersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "product_id")
	input.Filters.SortSafelist = []string{"product_id"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	userId := app.contextGetUser(r).ID
	orders, metadata, err := app.models.Orders.GetAllOrdersForUser(userId, input.Filters, r)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Include the metadata in the response envelope.
	err = app.writeJSON(w, http.StatusOK, envelope{"orders": orders, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the movie ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Delete the movie from the database, sending a 404 Not Found response to the
	// client if there isn't a matching record.
	err = app.models.Orders.Delete(id, r)
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
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "order successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateOrderHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	order, err := app.models.Orders.Get(id, r)
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
		Status     *int     `json:"status"`
		Address    *string  `json:"address"`
		TotalPrice *float32 `json:"totalPrice"`
	}
	// Decode the JSON as normal.
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if input.Status != nil {
		order.Status = *input.Status
	}
	// We also do the same for the other fields in the input struct.
	if input.Address != nil {
		order.Address = *input.Address
	}
	if input.TotalPrice != nil {
		order.TotalPrice = *input.TotalPrice
	}
	v := validator.New()
	if data.ValidateUpdatedOrder(v, order); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Orders.Update(order, r)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"order": order}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
