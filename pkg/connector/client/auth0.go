package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

type Client struct {
	wrapper     *uhttp.BaseHttpClient
	BearerToken string //nolint:gosec // intentional: stores the OAuth bearer token for API authentication
	BaseUrl     *url.URL
}

type ReqOpt func(reqURL *url.URL)

func WithQueryParam(key string, value string) ReqOpt {
	return func(reqURL *url.URL) {
		q := reqURL.Query()
		q.Set(key, value)
		reqURL.RawQuery = q.Encode()
	}
}

func New(
	ctx context.Context,
	baseUrl string,
	clientId string,
	clientSecret string,
) (*Client, error) {
	httpClient, err := uhttp.NewClient(
		ctx,
		uhttp.WithLogger(
			true,
			ctxzap.Extract(ctx),
		),
	)
	if err != nil {
		return nil, err
	}

	wrapper, err := uhttp.NewBaseHttpClientWithContext(ctx, httpClient)
	if err != nil {
		return nil, err
	}

	baseUrl0, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	client := Client{
		wrapper: wrapper,
		BaseUrl: baseUrl0,
	}

	err = client.Authorize(ctx, clientId, clientSecret)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (c *Client) Authorize(
	ctx context.Context,
	clientId string,
	clientSecret string,
) error {
	var target AuthResponse
	form := &url.Values{}
	form.Set("audience", c.BaseUrl.JoinPath(apiPathBase).String())
	form.Set("client_id", clientId)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")

	options := []uhttp.RequestOption{
		uhttp.WithFormBody(form.Encode()),
	}

	url := c.BaseUrl.JoinPath(apiPathAuth)
	request, err := c.wrapper.NewRequest(ctx, http.MethodPost, url, options...)
	if err != nil {
		return err
	}

	response, err := c.wrapper.Do(
		request,
		uhttp.WithJSONResponse(&target),
	)
	if err != nil {
		return fmt.Errorf("error authorizing: %w", err)
	}

	defer response.Body.Close()
	c.BearerToken = target.AccessToken
	return nil
}

func (c *Client) List(
	ctx context.Context,
	path string,
	target interface{},
	opts ...ReqOpt,
) (
	*v2.RateLimitDescription,
	error,
) {
	response, rateLimitData, err := c.get(
		ctx,
		path,
		&target,
		opts,
	)

	if err != nil {
		return rateLimitData, fmt.Errorf("error listing resource: %w", err)
	}

	defer response.Body.Close()

	return rateLimitData, nil
}

func (c *Client) GetUsers(
	ctx context.Context,
	limit int,
	page int,
	since string,
	until string,
) (
	[]User,
	int,
	*v2.RateLimitDescription,
	error,
) {
	var target UsersResponse
	rateLimitData, err := c.List(
		ctx,
		apiPathGetUsers,
		&target,
		WithQueryParam("include_totals", "true"),
		WithQueryParam("page", strconv.Itoa(page)),
		WithQueryParam("per_page", strconv.Itoa(limit)),
		WithQueryParam("q=created_at", fmt.Sprintf("[%s TO %s]", since, until)),
	)
	if err != nil {
		return nil, 0, rateLimitData, err
	}

	return target.Users, target.Total, rateLimitData, nil
}

func (c *Client) GetRoles(
	ctx context.Context,
	limit int,
	page int,
) (
	[]Role,
	int,
	*v2.RateLimitDescription,
	error,
) {
	var target RolesResponse
	rateLimitData, err := c.List(
		ctx,
		apiPathGetRoles,
		&target,
		WithQueryParam("page", strconv.Itoa(page)),
		WithQueryParam("per_page", strconv.Itoa(limit)),
	)
	if err != nil {
		return nil, 0, rateLimitData, err
	}

	return target.Roles, target.Total, rateLimitData, nil
}

func (c *Client) GetOrganizations(
	ctx context.Context,
	limit int,
	page int,
) (
	[]Organization,
	int,
	*v2.RateLimitDescription,
	error,
) {
	var target OrganizationsResponse
	rateLimitData, err := c.List(
		ctx,
		apiPathGetOrganizations,
		&target,
		WithQueryParam("page", strconv.Itoa(page)),
		WithQueryParam("per_page", strconv.Itoa(limit)),
	)
	if err != nil {
		return nil, 0, rateLimitData, err
	}

	return target.Organizations, target.Total, rateLimitData, nil
}

func (c *Client) GetOrganizationMembers(
	ctx context.Context,
	organizationId string,
	limit int,
	page int,
) (
	[]User,
	int,
	*v2.RateLimitDescription,
	error,
) {
	var target OrganizationMembersResponse
	rateLimitData, err := c.List(
		ctx,
		fmt.Sprintf(apiPathOrganizationMembers, organizationId),
		&target,
		WithQueryParam("page", strconv.Itoa(page)),
		WithQueryParam("per_page", strconv.Itoa(limit)),
	)
	if err != nil {
		return nil, 0, rateLimitData, err
	}

	return target.Members, target.Total, rateLimitData, nil
}

func (c *Client) GetRoleUsers(
	ctx context.Context,
	roleId string,
	limit int,
	page int,
) (
	[]User,
	int,
	*v2.RateLimitDescription,
	error,
) {
	var target RolesUsersResponse
	rateLimitData, err := c.List(
		ctx,
		fmt.Sprintf(apiPathUsersForRole, roleId),
		&target,
		WithQueryParam("page", strconv.Itoa(page)),
		WithQueryParam("per_page", strconv.Itoa(limit)),
	)
	if err != nil {
		return nil, 0, rateLimitData, err
	}

	return target.Users, target.Total, rateLimitData, nil
}

func (c *Client) AddUserToRole(
	ctx context.Context,
	roleId string,
	userId string,
) (
	*v2.RateLimitDescription,
	error,
) {
	var target RolesUsersResponse
	response, rateLimitData, err := c.post(
		ctx,
		fmt.Sprintf(apiPathRolesForUser, userId),
		map[string]interface{}{
			"roles": []string{roleId},
		},
		&target,
	)
	if err != nil {
		return rateLimitData, err
	}
	defer response.Body.Close()
	// TODO MARCOS check for double grant.
	return rateLimitData, nil
}

func (c *Client) RemoveUserFromRole(
	ctx context.Context,
	roleId string,
	userId string,
) (
	*v2.RateLimitDescription,
	error,
) {
	var target RolesUsersResponse
	response, rateLimitData, err := c.delete(
		ctx,
		fmt.Sprintf(apiPathRolesForUser, userId),
		map[string]interface{}{
			"roles": []string{roleId},
		},
		&target,
	)
	if err != nil {
		return rateLimitData, err
	}

	defer response.Body.Close()
	// TODO MARCOS check for double revoke.
	return rateLimitData, nil
}

func (c *Client) AddUserToOrganization(
	ctx context.Context,
	organizationId string,
	userId string,
) (
	*v2.RateLimitDescription,
	error,
) {
	response, rateLimitData, err := c.postNoJSONResponse(
		ctx,
		fmt.Sprintf(apiPathOrganizationMembers, organizationId),
		map[string]interface{}{
			"members": []string{userId},
		},
	)
	if err != nil {
		return rateLimitData, err
	}

	defer response.Body.Close()
	// TODO: MARCOS check for double grant.
	return rateLimitData, nil
}

func (c *Client) RemoveUserFromOrganization(
	ctx context.Context,
	organizationId string,
	userId string,
) (
	*v2.RateLimitDescription,
	error,
) {
	response, rateLimitData, err := c.deleteNoJSONResponse(
		ctx,
		fmt.Sprintf(apiPathOrganizationMembers, organizationId),
		map[string]interface{}{
			"members": []string{userId},
		},
	)
	if err != nil {
		return rateLimitData, fmt.Errorf("error removing user from organization: %w", err)
	}

	defer response.Body.Close()
	// TODO MARCOS check for double revoke.
	return rateLimitData, nil
}

func (c *Client) GetResourceServers(
	ctx context.Context,
	limit int,
	page int,
) (
	[]*ResourceServer,
	int,
	*v2.RateLimitDescription,
	error,
) {
	var target ResourceServerResponse
	rateLimitData, err := c.List(
		ctx,
		apiPathGetResourceServers,
		&target,
		WithQueryParam("page", strconv.Itoa(page)),
		WithQueryParam("per_page", strconv.Itoa(limit)),
	)
	if err != nil {
		return nil, 0, rateLimitData, err
	}

	return target.ResourceServers, target.Total, rateLimitData, nil
}

func (c *Client) GetResourceServer(
	ctx context.Context,
	id string,
) (
	*ResourceServer,
	*v2.RateLimitDescription,
	error,
) {
	var target ResourceServer
	response, rateLimitData, err := c.get(
		ctx,
		fmt.Sprintf(apiPathResourceServers, id),
		&target,
		nil,
	)
	if err != nil {
		return nil, rateLimitData, err
	}

	defer response.Body.Close()

	return &target, rateLimitData, nil
}

func (c *Client) GetRolePermissions(
	ctx context.Context,
	id string,
) (
	[]*RolePermission,
	*v2.RateLimitDescription,
	error,
) {
	var target []*RolePermission
	response, rateLimitData, err := c.get(
		ctx,
		fmt.Sprintf(apiPathRolePermissions, id),
		&target,
		nil,
	)
	if err != nil {
		return nil, rateLimitData, err
	}

	defer response.Body.Close()

	return target, rateLimitData, nil
}
