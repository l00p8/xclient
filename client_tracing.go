package http_client

import (
	"context"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/gojektech/heimdall/hystrix"
	"github.com/l00p8/tracer"
	"go.opentelemetry.io/otel/codes"
)

func WithTracing(service string, cfg Config) Client {
	base := cfg.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClt := &http.Client{
		Transport: otelhttp.NewTransport(base),
		Timeout:   time.Duration(cfg.HttpTimeout) * time.Millisecond,
	}
	clt := newHystrixClient(service, cfg, httpClt)
	return &clientWithTracing{
		clt: clt,
	}
}

type clientWithTracing struct {
	clt *hystrix.Client
}

func (clt *clientWithTracing) Request(ctx context.Context, method string, url string, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		tracer.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if headers == nil {
		headers = make(http.Header)
	}
	req.Header = headers
	req = tracer.Inject(ctx, req)

	resp, err := clt.clt.Do(req)
	if err != nil {
		tracer.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		return nil, err
	}
	tracer.SpanFromContext(ctx).SetStatus(codes.Ok, "OK")
	return resp, nil
}