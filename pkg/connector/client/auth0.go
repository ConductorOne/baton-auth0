package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

type Client struct {
	wrapper     *uhttp.BaseHttpClient
	BearerToken string
	BaseUrl     *url.URL
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
	limit int,
	page int,
) (
	*v2.RateLimitDescription,
	error,
) {
	response, rateLimitData, err := c.get(
		ctx,
		path,
		map[string]interface{}{
			// Note: `include_totals` changes the shape of the response
			"include_totals": true,
			"page":           page,
			"per_page":       limit,
		},
		&target,
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
		limit,
		page,
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
		limit,
		page,
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
		limit,
		page,
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
		limit,
		page,
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
		limit,
		page,
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
		limit,
		page,
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
		nil,
		&target,
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
		nil,
		&target,
	)
	if err != nil {
		return nil, rateLimitData, err
	}

	defer response.Body.Close()

	return target, rateLimitData, nil
}

func (c *Client) CreateJob(
	ctx context.Context,
	limit int,
) (
	*Job,
	*v2.RateLimitDescription,
	error,
) {
	bodyJSON := struct {
		Format string     `json:"format"`
		Limit  int        `json:"limit"`
		Fields []JobField `json:"fields"`
	}{
		Format: "json",
		Limit:  limit,
		Fields: []JobField{
			{
				Name: "user_id",
			},
			{
				Name: "name",
			},
			{
				Name: "email",
			},
			{
				Name: "nickname",
			},
			{
				Name: "created_at",
			},
			{
				Name: "updated_at",
			},
		},
	}

	var target Job

	response, rateLimitData, err := c.post(
		ctx,
		apiPathCreateUsersJob,
		bodyJSON,
		&target,
	)
	if err != nil {
		return nil, rateLimitData, err
	}

	defer response.Body.Close()

	return &target, rateLimitData, nil
}

func (c *Client) GetJob(
	ctx context.Context,
	id string,
) (
	*Job,
	*v2.RateLimitDescription,
	error,
) {
	var target Job

	path := c.getUrl(fmt.Sprintf(apiPathGetJob, id), make(map[string]interface{})).String()

	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.BearerToken))

	response, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("error getting job: status code %d", response.StatusCode)
	}

	err = json.NewDecoder(response.Body).Decode(&target)
	if err != nil {
		return nil, nil, err
	}

	return &target, nil, nil
}
