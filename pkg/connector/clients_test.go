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
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/stretchr/testify/require"
)

func TestClientsList(t *testing.T) {
	ctx := context.Background()

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
			"start": 0,
			"limit": 100,
			"total": 2,
			"clients": []map[string]interface{}{
				{
					"client_id":   "m2m-abc",
					"name":        "Backend Service",
					"app_type":    "non_interactive",
					"grant_types": []string{"client_credentials"},
				},
				{
					// app_type empty but client_credentials present -> still M2M via fallback.
					"client_id":   "m2m-def",
					"name":        "Legacy Service",
					"app_type":    "",
					"grant_types": []string{"client_credentials"},
				},
			},
		})
	}))
	defer server.Close()

	c0, err := client2.New(ctx, server.URL, "mock", "token")
	require.Nil(t, err)

	b := newClientBuilder(c0)

	resources, _, _, err := b.List(ctx, nil, &pagination.Token{Size: 100})
	require.Nil(t, err)
	require.Len(t, resources, 2)

	for _, res := range resources {
		require.Equal(t, "client", res.Id.ResourceType)

		nhi, err := resourceSdk.GetNonHumanIdentityTrait(res)
		require.Nil(t, err, "expected NHI trait on M2M client resource")
		require.Equal(t, v2.NonHumanIdentityTrait_NHI_TYPE_APP_REGISTRATION, nhi.GetNhiType())
		require.Equal(t, "auth0.m2m_client", nhi.GetNhiDetail())
	}
}

func TestApplicationIsM2M(t *testing.T) {
	cases := []struct {
		name string
		app  client2.Application
		want bool
	}{
		{"non_interactive app_type", client2.Application{AppType: "non_interactive"}, true},
		{"client_credentials grant fallback", client2.Application{AppType: "regular_web", GrantTypes: []string{"client_credentials"}}, true},
		{"human-facing spa", client2.Application{AppType: "spa", GrantTypes: []string{"authorization_code"}}, false},
		{"native app", client2.Application{AppType: "native"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.app.IsM2M())
		})
	}
}
