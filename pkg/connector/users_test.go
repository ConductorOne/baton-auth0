package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	"github.com/conductorone/baton-auth0/test"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestUsersListMaxResultsCap(t *testing.T) {
	ctx := context.Background()

	t.Run("should stop pagination at Auth0UserSearchMaxResults", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if strings.Contains(r.URL.String(), "oauth/token") {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "mock-token",
					"token_type":   "Bearer",
					"expires_in":   86400,
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"start":  0,
				"limit":  100,
				"length": 1,
				"total":  1500,
				"users": []map[string]interface{}{{
					"user_id":        "auth0|test123",
					"email":          "test@example.com",
					"name":           "Test User",
					"nickname":       "testuser",
					"email_verified": false,
					"created_at":     "2024-01-01T00:00:00.000Z",
					"updated_at":     "2024-01-01T00:00:00.000Z",
					"identities":     []interface{}{},
				}},
			})
		}))
		defer server.Close()

		c0, err := client.New(ctx, server.URL, "mock", "token")
		require.Nil(t, err)

		ub := newUserBuilder(c0)

		// Page 0, limit 100: total is capped to 1000, next token expected (100 < 1000).
		pToken := &pagination.Token{Token: "", Size: 100}
		_, nextToken, _, err := ub.List(ctx, nil, pToken)
		require.Nil(t, err)
		require.NotEmpty(t, nextToken, "expected a next token before reaching the 1000 cap")

		// Page 9, limit 100: (9+1)*100 = 1000 >= 1000, no next token expected.
		page9Bytes, _ := json.Marshal(client.Pagination{Page: 9})
		pToken9 := &pagination.Token{Token: string(page9Bytes), Size: 100}
		_, nextToken9, _, err := ub.List(ctx, nil, pToken9)
		require.Nil(t, err)
		require.Empty(t, nextToken9, "expected no next token at the 1000 user cap boundary")
	})
}

func TestUsersList(t *testing.T) {
	ctx := context.Background()

	t.Run("should get users with pagination", func(t *testing.T) {
		server := test.FixturesServer()
		defer server.Close()

		percipioClient, err := client.New(
			ctx,
			server.URL,
			"mock",
			"token",
		)
		if err != nil {
			t.Fatal(err)
		}

		c := newUserBuilder(percipioClient)

		resources := make([]*v2.Resource, 0)
		pToken := pagination.Token{
			Token: "",
			Size:  1,
		}

		for i := 0; i < 2; i++ {
			nextResources, nextToken, listAnnotations, err := c.List(ctx, nil, &pToken)
			resources = append(resources, nextResources...)

			require.Nil(t, err)
			test.AssertNoRatelimitAnnotations(t, listAnnotations)

			if nextToken == "" {
				break
			}

			var token client.Pagination
			err = json.Unmarshal([]byte(nextToken), &token)
			require.Nil(t, err)
			require.Equal(t, token.Page, i+1)

			pToken.Token = nextToken
		}

		require.NotNil(t, resources)
		require.Len(t, resources, 2)
		require.NotEmpty(t, resources[0].Id)
	})
}
