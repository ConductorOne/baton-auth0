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

func TestRolesGrantsRepeatedNextTokenTerminates(t *testing.T) {
	ctx := context.Background()
	permissionsCalls := 0
	usersCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if mockTokenResponse(w, r) {
			return
		}
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.Path, "/permissions") {
			permissionsCalls++
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		usersCalls++
		// Every call returns the same user plus the SAME next cursor.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []map[string]interface{}{
				{"user_id": "auth0|user-1"},
			},
			"next": "cursor-A",
		})
	}))
	defer server.Close()

	// Helper unit assertions for the guard.
	require.Equal(t, "", client2.GetNextRoleUserCheckpointToken("cursor-A", "cursor-A"))
	require.NotEqual(t, "", client2.GetNextRoleUserCheckpointToken("cursor-A", "cursor-B"))
	require.Equal(t, "", client2.GetNextRoleUserCheckpointToken("", ""))
	require.Equal(t, "", client2.GetNextRoleUserCheckpointToken("cursor-A", ""))

	c0, err := client2.New(ctx, server.URL, "mock", "token")
	require.Nil(t, err)

	role, err := roleResource(client2.Role{ID: "role-1", Name: "Test Role"}, nil)
	require.Nil(t, err)

	rb := newRoleBuilder(c0, true)
	pToken := &pagination.Token{Token: ""}

	const callCap = 5
	finalToken := ""
	for range callCap {
		_, nextToken, _, err := rb.Grants(ctx, role, pToken)
		require.Nil(t, err)
		if nextToken == "" {
			finalToken = ""
			break
		}
		finalToken = nextToken
		pToken = &pagination.Token{Token: nextToken}
	}
	require.Equal(t, "", finalToken, "walk must terminate despite a repeated next token")
	require.LessOrEqual(t, usersCalls, callCap)
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

func TestRolesGrantsMultiPageWalkUnchanged(t *testing.T) {
	ctx := context.Background()
	usersCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if mockTokenResponse(w, r) {
			return
		}
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.Path, "/permissions") {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"permission_name":           "read:things",
					"resource_server_identifier": "api-1",
				},
			})
			return
		}
		usersCalls++
		switch usersCalls {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"users": []map[string]interface{}{
					{"user_id": "auth0|user-1"},
				},
				"next": "cursor-A",
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"users": []map[string]interface{}{
					{"user_id": "auth0|user-2"},
				},
				"next": "",
			})
		}
	}))
	defer server.Close()

	c0, err := client2.New(ctx, server.URL, "mock", "token")
	require.Nil(t, err)

	role, err := roleResource(client2.Role{ID: "role-1", Name: "Test Role"}, nil)
	require.Nil(t, err)

	rb := newRoleBuilder(c0, true)
	pToken := &pagination.Token{Token: ""}

	var allGrants []*v2.Grant
	finalToken := ""
	for range 5 {
		grants, nextToken, _, err := rb.Grants(ctx, role, pToken)
		require.Nil(t, err)
		allGrants = append(allGrants, grants...)
		finalToken = nextToken
		if nextToken == "" {
			break
		}
		pToken = &pagination.Token{Token: nextToken}
	}
	require.Equal(t, "", finalToken)
	require.Equal(t, 2, usersCalls, "both pages must be fetched")
	principals := map[string]bool{}
	for _, g := range allGrants {
		principals[g.Principal.Id.Resource] = true
	}
	require.True(t, principals["auth0|user-1"], "page 1 grant must be collected")
	require.True(t, principals["auth0|user-2"], "page 2 grant must be collected")
}