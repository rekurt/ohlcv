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
	"context"
	"net/http"
)

// MarketApiRouter defines the required methods for binding the api requests to a responses for the MarketApi
// The MarketApiRouter implementation should parse necessary information from the http request,
// pass the data to a MarketApiServicer to perform the required actions, then write the service results to the http response.
type MarketApiRouter interface {
	ApiV1TradesGet(http.ResponseWriter, *http.Request)
}

// MarketApiServicer defines the api actions for the MarketApi service
// This interface intended to stay up to date with the openapi yaml used to generate it,
// while the service implementation can be ignored with the .openapi-generator-ignore file
// and updated with the logic required for the API.
type MarketApiServicer interface {
	ApiV1TradesGet(context.Context, string, int32) (ImplResponse, error)
}
