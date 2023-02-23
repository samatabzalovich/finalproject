package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	// Use the requirePermission() middleware on each of the /v1/movies** endpoints,
	// passing in the required permission code as the first parameter.
	router.HandlerFunc(http.MethodGet, "/v1/users/orders", app.requirePermission("products:read", app.listUserOrdersHandler))     //
	router.HandlerFunc(http.MethodPost, "/v1/users/orders", app.requirePermission("products:order", app.orderProductHandler))     //
	router.HandlerFunc(http.MethodDelete, "/v1/users/orders/:id", app.requirePermission("products:read", app.deleteOrderHandler)) //
	router.HandlerFunc(http.MethodPut, "/v1/users/orders/:id", app.requirePermission("products:write", app.updateOrderHandler))   //
	router.HandlerFunc(http.MethodGet, "/v1/products", app.requirePermission("products:read", app.listProductsHandler))           //
	router.HandlerFunc(http.MethodPost, "/v1/products", app.requirePermission("products:write", app.createProductHandler))        //
	router.HandlerFunc(http.MethodGet, "/v1/products/:id", app.requirePermission("products:read", app.showProductHandler))        //
	router.HandlerFunc(http.MethodPatch, "/v1/products/:id", app.requirePermission("products:write", app.updateProductHandler))   //
	router.HandlerFunc(http.MethodDelete, "/v1/products/:id", app.requirePermission("products:write", app.deleteProductHandler))  //
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)                                                     //
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)                                            //
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)                        //
	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
