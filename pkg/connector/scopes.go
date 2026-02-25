package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

var _ connectorbuilder.ResourceSyncer = (*scopeBuilder)(nil)

type scopeBuilder struct {
	client *client.Client
}

func newScopeBuilder(client *client.Client) *scopeBuilder {
	return &scopeBuilder{client: client}
}

func (r *scopeBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return scopeResourceType
}

func (r *scopeBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	page, limit, _, _, _, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}
	var outputAnnotations annotations.Annotations

	resourcesServer, total, rateLimitData, err := r.client.GetResourceServers(ctx, limit, page)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	if len(resourcesServer) == 0 {
		return nil, "", outputAnnotations, nil
	}

	outputResources := make([]*v2.Resource, 0, len(resourcesServer))
	for _, rsServer := range resourcesServer {
		for _, scope := range rsServer.Scopes {
			scopeRs, err := scopeResource(scope, rsServer)
			if err != nil {
				return nil, "", nil, err
			}
			outputResources = append(outputResources, scopeRs)
		}
	}

	nextToken := client.GetNextToken(page, limit, total, nil)

	return outputResources, nextToken, outputAnnotations, nil
}

func (r *scopeBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (r *scopeBuilder) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func scopeResource(resourceServer client.ResourceServerScope, server *client.ResourceServer) (*v2.Resource, error) {
	// Needs to be in the format of <resourceServerId>/<scopeName>
	// Since each scope is unique to a resource server, we can use the resource server ID as a prefix
	scopeId := formatScopeId(resourceServer, server)
	scopeName := fmt.Sprintf("%s/%s", server.Name, resourceServer.Value)

	resource0, err := resource.NewResource(
		scopeName,
		scopeResourceType,
		scopeId,
		resource.WithDescription(resourceServer.Description),
		resource.WithParentResourceID(&v2.ResourceId{
			ResourceType: resourceServerResourceType.Id,
			Resource:     server.Id,
		}),
	)
	if err != nil {
		return nil, err
	}

	return resource0, nil
}

func formatScopeId(resourceServer client.ResourceServerScope, server *client.ResourceServer) string {
	return fmt.Sprintf("%s:%s", server.Identifier, resourceServer.Value)
}
