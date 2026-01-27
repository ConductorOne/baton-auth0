package connector

import (
	"context"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

var _ connectorbuilder.ResourceSyncer = (*resourceServerBuilder)(nil)

type resourceServerBuilder struct {
	client *client.Client
}

func newResourceServerBuilder(client *client.Client) *resourceServerBuilder {
	return &resourceServerBuilder{client: client}
}

func (r *resourceServerBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return resourceServerResourceType
}

func (r *resourceServerBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	page, limit, _, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}
	var outputAnnotations annotations.Annotations

	resourcesServer, total, ratelimitData, err := r.client.GetResourceServers(ctx, limit, page)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	if len(resourcesServer) == 0 {
		return nil, "", outputAnnotations, nil
	}

	outputResources := make([]*v2.Resource, 0, len(resourcesServer))
	for _, rsServer := range resourcesServer {
		organizationResource0, err := resourceServerResource(rsServer)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, organizationResource0)
	}

	nextToken := client.GetNextToken(page, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

func (r *resourceServerBuilder) Entitlements(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	ent := []*v2.Entitlement{
		entitlement.NewPermissionEntitlement(
			resource,
			"scope",
			entitlement.WithGrantableTo(scopeResourceType),
			entitlement.WithDisplayName("Scope"),
			entitlement.WithDescription("The scope of the resource server, which defines the permissions granted to the resource server."),
			entitlement.WithAnnotation(&v2.EntitlementImmutable{}),
		),
	}

	return ent, "", nil, nil
}

func (r *resourceServerBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var outputAnnotations annotations.Annotations
	server, ratelimitData, err := r.client.GetResourceServer(ctx, resource.Id.Resource)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", nil, err
	}

	grantsResponse := make([]*v2.Grant, 0, len(server.Scopes))
	for _, scope := range server.Scopes {
		newGrant := grant.NewGrant(resource, "scope", &v2.ResourceId{
			ResourceType: scopeResourceType.Id,
			Resource:     formatScopeId(scope, server),
		})
		grantsResponse = append(grantsResponse, newGrant)
	}

	return grantsResponse, "", nil, nil
}

func resourceServerResource(resourceServer *client.ResourceServer) (*v2.Resource, error) {
	resource0, err := resource.NewResource(
		resourceServer.Name,
		resourceServerResourceType,
		resourceServer.Id,
	)
	if err != nil {
		return nil, err
	}

	return resource0, nil
}
