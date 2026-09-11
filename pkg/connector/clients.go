package connector

import (
	"context"
	"strings"

	client2 "github.com/conductorone/baton-auth0/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
)

var _ connectorbuilder.ResourceSyncer = (*clientBuilder)(nil)

type clientBuilder struct {
	client *client2.Client
}

func newClientBuilder(client *client2.Client) *clientBuilder {
	return &clientBuilder{client: client}
}

func (b *clientBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return clientResourceType
}

// clientResource builds a resource for an Auth0 M2M client. The
// NonHumanIdentityTrait marks it as an APP_REGISTRATION (it holds its own
// credentials & scopes), with axis-2 detail "auth0.m2m_client".
func clientResource(app client2.Application) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"client_id":   app.ClientId,
		"app_type":    app.AppType,
		"grant_types": strings.Join(app.GrantTypes, ","),
	}

	appTraitOptions := []resourceSdk.AppTraitOption{
		resourceSdk.WithAppProfile(profile),
	}

	return resourceSdk.NewAppResource(
		app.Name,
		clientResourceType,
		app.ClientId,
		appTraitOptions,
		resourceSdk.WithNHIType(v2.NonHumanIdentityTrait_NHI_TYPE_APP_REGISTRATION, "auth0.m2m_client"),
	)
}

// List returns the tenant's machine-to-machine (M2M) clients. The Auth0
// Management API enumerates clients via GET /api/v2/clients; the client layer
// filters to app_type=non_interactive server-side, and we defensively skip any
// non-M2M client that slips through.
func (b *clientBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	page, limit, _, err := client2.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}
	var outputAnnotations annotations.Annotations

	apps, total, rateLimitData, err := b.client.GetClients(ctx, limit, page)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outputAnnotations, err
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	if len(apps) == 0 {
		return nil, "", outputAnnotations, nil
	}

	outputResources := make([]*v2.Resource, 0, len(apps))
	for _, app := range apps {
		if !app.IsM2M() {
			continue
		}
		resource0, err := clientResource(app)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource0)
	}

	nextToken := client2.GetNextToken(page, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for M2M clients.
func (b *clientBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for M2M clients.
func (b *clientBuilder) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}
