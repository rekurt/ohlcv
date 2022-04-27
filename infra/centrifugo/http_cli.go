package centrifugo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

const defaultAPITimeout = 4 * time.Second
const defaultRetryPostNum = 4

type restyPostClient struct {
	cli *resty.Client
}

func newRestyPostClient(baseURL, authKey string) *restyPostClient {
	client := resty.New()

	client.SetBaseURL("http//:" + baseURL)
	client.SetTimeout(defaultAPITimeout)
	client.SetRetryCount(defaultRetryPostNum)

	client.SetHeaders(
		map[string]string{
			headers.Authorization: "apikey " + authKey,
			headers.ContentType:   "application/json",
		},
	)

	return &restyPostClient{
		cli: client,
	}
}

func (r *restyPostClient) Post(
	ctx context.Context,
	body WSPubCommand,
) (json.RawMessage, error) {
	req := r.cli.R()
	req.SetContext(ctx)
	req.SetBody(body)
	req.URL = r.cli.BaseURL

	resp, err := req.Send()
	if err != nil || resp.Error() != nil {
		return nil, errors.Wrap(err, "can't do POST request for "+r.cli.BaseURL)
	}

	return resp.Body(), nil
}
