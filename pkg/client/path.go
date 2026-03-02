package client

import (
	liburl "net/url"
	"strconv"
)

const (
	apiPathAuth                = "/oauth/token"
	apiPathBase                = "/api/v2/" // Note: trailing slash is required by audience.
	apiPathOrganizationMembers = "/api/v2/organizations/%s/members"
	apiPathGetOrganizations    = "/api/v2/organizations"
	apiPathGetRoles            = "/api/v2/roles"
	apiPathGetUsers            = "/api/v2/users"
	apiPathRolesForUser        = "/api/v2/users/%s/roles"
	apiPathUsersForRole        = "/api/v2/roles/%s/users"
	apiPathGetResourceServers  = "/api/v2/resource-servers"
	apiPathResourceServers     = "/api/v2/resource-servers/%s"
	apiPathRolePermissions     = "/api/v2/roles/%s/permissions"
)

func (c *Client) getUrl(
	path string,
	queryParameters map[string]interface{},
) *liburl.URL {
	params := liburl.Values{}
	for key, valueAny := range queryParameters {
		switch value := valueAny.(type) {
		case string:
			params.Add(key, value)
		case int:
			params.Add(key, strconv.Itoa(value))
		case bool:
			params.Add(key, strconv.FormatBool(value))
		default:
			continue
		}
	}

	output := c.BaseUrl.JoinPath(path)
	output.RawQuery = params.Encode()
	return output
}
