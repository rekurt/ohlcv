package market

import (
	"context"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

const (
	httpMethodList    = "GET"
	uriPathClientList = "/v1/internal/markets"
)

type Market struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	BaseCurrency               Currency `json:"base_currency"`
	QuotedCurrency             Currency `json:"quoted_currency"`
	MakerFee                   string   `json:"maker_fee"`
	TakerFee                   string   `json:"taker_fee"`
	Precision                  int64    `json:"precision"`
	BasePrecision              int64    `json:"base_precision"`
	QuotedPrecision            int64    `json:"quoted_precision"`
	OrderMinAmount             string   `json:"order_min_amount"`
	OrderMinPrice              string   `json:"order_min_price"`
	OrderMinSize               string   `json:"order_min_size"`
	OrderPriceDeviationPercent string   `json:"order_price_deviation_percent"`
	IsVolatile                 bool     `json:"is_volatile"`
}

// Currency ...
type Currency struct {
	ID        string `json:"id"`
	Symbol    string `json:"symbol"`
	Precision uint8  `json:"precision"`
}

type Client interface {
	List(ctx context.Context) (markets []Market, err error)
}

type client struct {
	cli           *fasthttp.HostClient
	transportList ListTransport
	options       map[interface{}]Option
	token         string
}

type option struct{}

type Option interface {
	Prepare(ctx context.Context, r *fasthttp.Request)
}

var (
	List = option{}
)

// List ...
func (s *client) List(ctx context.Context) (markets []Market, err error) {
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()
	if opt, ok := s.options[List]; ok {
		opt.Prepare(ctx, req)
	}
	if err = s.transportList.EncodeRequest(ctx, req, &s.token); err != nil {
		return
	}
	err = s.cli.Do(req, res)
	if err != nil {
		return
	}
	return s.transportList.DecodeResponse(ctx, res)
}

type Config struct {
	ServerURL           string
	ServerTLS           bool
	MaxConns            *int
	MaxConnDuration     *time.Duration
	MaxIdleConnDuration *time.Duration
	ReadBufferSize      *int
	WriteBufferSize     *int
	ReadTimeout         *time.Duration
	WriteTimeout        *time.Duration
	MaxResponseBodySize *int
}

func New(
	config Config,
	errorProcessor errorProcessor,
	options map[interface{}]Option,
	token string,
) (client Client, err error) {
	parsedServerURL, err := url.Parse(config.ServerURL)
	if err != nil {
		err = errors.Wrap(err, "failed to parse server url")
		return
	}
	transportList := NewListTransport(
		errorProcessor,
		parsedServerURL.Scheme+"://"+parsedServerURL.Host+parsedServerURL.Path+uriPathClientList,
		httpMethodList,
	)

	cli := fasthttp.HostClient{
		Addr: parsedServerURL.Host,
	}
	if config.MaxConns != nil {
		cli.MaxConns = *config.MaxConns
	}
	if config.MaxConnDuration != nil {
		cli.MaxConnDuration = *config.MaxConnDuration
	}
	if config.MaxIdleConnDuration != nil {
		cli.MaxIdleConnDuration = *config.MaxIdleConnDuration
	}
	if config.ReadBufferSize != nil {
		cli.ReadBufferSize = *config.ReadBufferSize
	}
	if config.WriteBufferSize != nil {
		cli.WriteBufferSize = *config.WriteBufferSize
	}
	if config.ReadTimeout != nil {
		cli.ReadTimeout = *config.ReadTimeout
	}
	if config.WriteTimeout != nil {
		cli.WriteTimeout = *config.WriteTimeout
	}
	if config.MaxResponseBodySize != nil {
		cli.MaxResponseBodySize = *config.MaxResponseBodySize
	}

	cli.IsTLS = config.ServerTLS

	client = newClient(
		&cli,
		transportList,
		options,
		token,
	)
	return
}

func newClient(
	cli *fasthttp.HostClient,
	transportList ListTransport,
	options map[interface{}]Option,
	token string,
) Client {
	return &client{
		cli:           cli,
		transportList: transportList,
		options:       options,
		token:         token,
	}
}
