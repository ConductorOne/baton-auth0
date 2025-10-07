package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// WithBearerToken - TODO(marcos): move this function to `baton-sdk`.
func WithBearerToken(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

func (c *Client) get(
	ctx context.Context,
	path string,
	queryParameters map[string]interface{},
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequest(
		ctx,
		http.MethodGet,
		path,
		queryParameters,
		nil,
		target,
		true,
	)
}

func (c *Client) post(
	ctx context.Context,
	path string,
	body interface{},
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequest(
		ctx,
		http.MethodPost,
		path,
		nil,
		body,
		target,
		true,
	)
}

func (c *Client) postNoJSONResponse(
	ctx context.Context,
	path string,
	body interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequestNoJSONResponse(
		ctx,
		http.MethodPost,
		path,
		nil,
		body,
	)
}

func (c *Client) delete(
	ctx context.Context,
	path string,
	body interface{},
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequest(
		ctx,
		http.MethodDelete,
		path,
		nil,
		body,
		target,
		true,
	)
}

func (c *Client) deleteNoJSONResponse(
	ctx context.Context,
	path string,
	body interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequestNoJSONResponse(
		ctx,
		http.MethodDelete,
		path,
		nil,
		body,
	)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	queryParameters map[string]interface{},
	payload interface{},
	target interface{},
	cache bool,
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	options := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		WithBearerToken(c.BearerToken),
	}
	if payload != nil {
		options = append(options, uhttp.WithJSONBody(payload))
	}

	if !cache {
		options = append(options, uhttp.WithNoCache())
	}

	url := c.getUrl(path, queryParameters)

	request, err := c.wrapper.NewRequest(ctx, method, url, options...)
	if err != nil {
		return nil, nil, err
	}
	var ratelimitData v2.RateLimitDescription
	response, err := c.wrapper.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
		uhttp.WithJSONResponse(target),
	)

	if err != nil {
		if response != nil {
			return nil, &ratelimitData, fmt.Errorf("error doing request: %w, body: %v", err, logBody(response.Body))
		}
		return nil, &ratelimitData, fmt.Errorf("error doing request: %w", err)
	}

	return response, &ratelimitData, nil
}

func logBody(body io.ReadCloser) string {
	var out = []byte("")
	if body == nil {
		return string(out)
	}
	defer body.Close()
	out, _ = io.ReadAll(body)
	return string(out)
}

func (c *Client) doRequestNoJSONResponse(
	ctx context.Context,
	method string,
	path string,
	queryParameters map[string]interface{},
	payload interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	options := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		WithBearerToken(c.BearerToken),
	}
	if payload != nil {
		options = append(options, uhttp.WithJSONBody(payload))
	}

	url := c.getUrl(path, queryParameters)

	request, err := c.wrapper.NewRequest(ctx, method, url, options...)
	if err != nil {
		return nil, nil, err
	}

	var ratelimitData v2.RateLimitDescription
	response, err := c.wrapper.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
	)

	if err != nil {
		return nil, &ratelimitData, fmt.Errorf("error doing request: %w, body: %v", err, logBody(response.Body))
	}

	return response, &ratelimitData, nil
}
