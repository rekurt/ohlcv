/*
 * PointPay.io Public Spot API (draft)
 *
 * OpenAPI Specifications for the PointPay.io Public Spot API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"net/http"
	"strings"
)

// MarketApiController binds http requests to an api service and writes the service results to the http response
type MarketApiController struct {
	service      MarketApiServicer
	errorHandler ErrorHandler
}

// MarketApiOption for how the controller is set up.
type MarketApiOption func(*MarketApiController)

// WithMarketApiErrorHandler inject ErrorHandler into controller
func WithMarketApiErrorHandler(h ErrorHandler) MarketApiOption {
	return func(c *MarketApiController) {
		c.errorHandler = h
	}
}

// NewMarketApiController creates a default api controller
func NewMarketApiController(s MarketApiServicer, opts ...MarketApiOption) Router {
	controller := &MarketApiController{
		service:      s,
		errorHandler: DefaultErrorHandler,
	}

	for _, opt := range opts {
		opt(controller)
	}

	return controller
}

// Routes returns all the api routes for the MarketApiController
func (c *MarketApiController) Routes() Routes {
	return Routes{
		{
			"ApiV1TradesGet",
			strings.ToUpper("Get"),
			"/api/v1/trades",
			c.ApiV1TradesGet,
		},
		{
			"ApiV3Ticker24hrGet",
			strings.ToUpper("Get"),
			"/api/v3/ticker/24hr",
			c.ApiV3Ticker24hrGet,
		},
		{
			"V1TradingStats24hAllGet",
			strings.ToUpper("Get"),

			"/v1/trading/stats/24h/all",
			c.V1TradingStats24hAllGet,
		},
	}
}

// ApiV1TradesGet - Recent Trades List
func (c *MarketApiController) ApiV1TradesGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	symbolParam := query.Get("symbol")
	limitParam, err := parseInt32Parameter(query.Get("limit"), false)
	if err != nil {
		c.errorHandler(w, r, &ParsingError{Err: err}, nil)
		return
	}
	result, err := c.service.ApiV1TradesGet(r.Context(), symbolParam, limitParam)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}

// ApiV3Ticker24hrGet - 24hr Ticker Price Change Statistics
func (c *MarketApiController) ApiV3Ticker24hrGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	symbolParam := query.Get("symbol")
	result, err := c.service.ApiV3Ticker24hrGet(r.Context(), symbolParam)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}

// V1TradingStats24hAllGet - 24hr Ticker Price Change Statistics With Market Info
func (c *MarketApiController) V1TradingStats24hAllGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	symbolParam := query.Get("symbol")
	result, err := c.service.V1TradingStats24hAllGet(r.Context(), symbolParam)
	// If an error occurred, encode the error with the status code
	if err != nil {
		c.errorHandler(w, r, err, &result)
		return
	}
	// If no error, encode the body and the result code
	EncodeJSONResponse(result.Body, &result.Code, w)

}
