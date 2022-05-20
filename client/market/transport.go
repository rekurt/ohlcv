package market

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"
)

type listResponse []Market

// ListTransport transport interface
type ListTransport interface {
	EncodeRequest(ctx context.Context, r *fasthttp.Request, token *string) (err error)
	DecodeResponse(ctx context.Context, r *fasthttp.Response) (markets []Market, err error)
}

type listTransport struct {
	errorProcessor errorProcessor
	pathTemplate   string
	method         string
}

// EncodeRequest method for decoding requests on server side
func (t *listTransport) EncodeRequest(ctx context.Context, r *fasthttp.Request, token *string) (err error) {
	r.Header.SetMethod(t.method)
	r.SetRequestURI(t.pathTemplate)

	_ = r.URI()

	r.Header.Set("Authorization", *token)

	r.Header.Set("Content-Type", "application/json")

	return
}

// DecodeResponse method for decoding response on server side
func (t *listTransport) DecodeResponse(ctx context.Context, r *fasthttp.Response) (markets []Market, err error) {
	if r.StatusCode() != 200 {
		err = t.errorProcessor.Decode(r)
		return
	}

	var theResponse listResponse
	if err = json.Unmarshal(r.Body(), &theResponse); err != nil {
		return
	}

	markets = theResponse

	for i := range markets {
		markets[i].Name = flipMarkets(markets[i].Name, "/")
	}

	return
}

func flipMarkets(s string, sep string) string {
	index := strings.Index(s, sep)
	if index == -1 {
		return s
	}
	return s[index+1:] + "_" + s[:index]
}

// NewListTransport the transport creator for http requests
func NewListTransport(
	errorProcessor errorProcessor,
	pathTemplate string,
	method string,
) ListTransport {
	return &listTransport{
		errorProcessor: errorProcessor,
		pathTemplate:   pathTemplate,
		method:         method,
	}
}

func ptr(in []byte) *string {
	i := string(in)
	return &i
}
