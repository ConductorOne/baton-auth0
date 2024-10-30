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
		&target,
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
		&target,
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
		&target,
	)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	queryParameters map[string]interface{},
	payload interface{},
	target interface{},
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
		uhttp.WithJSONResponse(target),
	)
	if err != nil {
		body, _ := io.ReadAll(response.Body)
		fmt.Println(string(body))
		return nil, &ratelimitData, err
	}
	defer response.Body.Close()

	return response, &ratelimitData, nil
}
