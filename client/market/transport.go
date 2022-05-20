package market

import (
	"context"
	"encoding/json"
	"fmt"
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
		split := strings.Split(markets[i].Name, "/")
		if len(split) != 2 {
			return nil, fmt.Errorf("unexpected market:%s", markets[i].Name)
		}
		if strings.TrimSpace(split[0]) == "" || strings.TrimSpace(split[1]) == "" {
			return nil, fmt.Errorf("unexpected market:%s", markets[i].Name)
		}

		markets[i].Name = split[1] + "_" + split[0]
	}

	return
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
