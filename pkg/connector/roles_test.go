package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client2 "github.com/conductorone/baton-auth0/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func mockTokenResponse(w http.ResponseWriter, r *http.Request) bool {
	if strings.Contains(r.URL.String(), "oauth/token") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token",
			"token_type":   "Bearer",
			"expires_in":   86400,
		})
		return true
	}
	return false
}

func TestRolesGrantsZeroPermissionsEmitsMembership(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if mockTokenResponse(w, r) {
			return
		}
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.Path, "/permissions") {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []map[string]interface{}{
				{"user_id": "auth0|user-1"},
			},
			"next": "",
		})
	}))
	defer server.Close()

	c0, err := client2.New(ctx, server.URL, "mock", "token")
	require.Nil(t, err)

	role, err := roleResource(client2.Role{ID: "role-1", Name: "Test Role"}, nil)
	require.Nil(t, err)

	rb := newRoleBuilder(c0, true)
	pToken := &pagination.Token{Token: ""}

	var allGrants []*v2.Grant
	for range 5 {
		grants, nextToken, _, err := rb.Grants(ctx, role, pToken)
		require.Nil(t, err)
		allGrants = append(allGrants, grants...)
		if nextToken == "" {
			break
		}
		pToken = &pagination.Token{Token: nextToken}
	}

	var membershipGrants []*v2.Grant
	for _, g := range allGrants {
		if strings.HasSuffix(g.Entitlement.Id, ":"+roleEntitlementName) {
			membershipGrants = append(membershipGrants, g)
		}
	}
	require.Len(t, membershipGrants, 1, "zero-permissions role must still emit its membership grant")
	require.Equal(t, role.Id.Resource, membershipGrants[0].Entitlement.Resource.Id.Resource)
	require.Equal(t, userResourceType.Id, membershipGrants[0].Principal.Id.ResourceType)
	require.Equal(t, "auth0|user-1", membershipGrants[0].Principal.Id.Resource)
}
