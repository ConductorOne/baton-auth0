package connector

import (
	"context"
	"fmt"

	client2 "github.com/conductorone/baton-auth0/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkEntitlement "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	sdkGrant "github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

var (
	_ connectorbuilder.ResourceSyncer      = (*organizationBuilder)(nil)
	_ connectorbuilder.ResourceProvisioner = (*organizationBuilder)(nil)
)

const organizationEntitlementName = "member"

type organizationBuilder struct {
	client *client2.Client
}

func (b *organizationBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return organizationResourceType
}

// Create a new connector resource for an Auth0 organization.
func organizationResource(
	organization client2.Organization,
	parentResourceID *v2.ResourceId,
) (*v2.Resource, error) {
	return resourceSdk.NewGroupResource(
		organization.Name,
		organizationResourceType,
		organization.ID,
		[]resourceSdk.GroupTraitOption{
			resourceSdk.WithGroupProfile(
				map[string]interface{}{
					"id":           organization.ID,
					"name":         organization.Name,
					"display_name": organization.DisplayName,
				},
			),
		},
		resourceSdk.WithParentResourceID(parentResourceID),
	)
}

// List returns all the organizations from the database as resource objects.
func (b *organizationBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	pToken *pagination.Token,
) (
	[]*v2.Resource,
	string,
	annotations.Annotations,
	error,
) {
	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	page, limit, _, err := client2.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	organizations, total, rateLimitData, err := b.client.GetOrganizations(ctx, limit, page)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outputAnnotations, err
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	if len(organizations) == 0 {
		return outputResources, "", outputAnnotations, nil
	}

	for _, organization := range organizations {
		organizationResource0, err := organizationResource(organization, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, organizationResource0)
	}

	nextToken := client2.GetNextToken(page, limit, total)
	return outputResources, nextToken, outputAnnotations, nil
}

func (b *organizationBuilder) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return []*v2.Entitlement{
		sdkEntitlement.NewAssignmentEntitlement(
			resource,
			organizationEntitlementName,
			sdkEntitlement.WithGrantableTo(userResourceType),
			sdkEntitlement.WithDisplayName(
				fmt.Sprintf("%s %s", resource.DisplayName, organizationEntitlementName),
			),
			sdkEntitlement.WithDescription(
				fmt.Sprintf("Member of %s organization in Auth0", resource.DisplayName),
			),
		),
	}, "", nil, nil
}

func (b *organizationBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	token *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	var outputAnnotations annotations.Annotations
	page, limit, _, err := client2.ParsePaginationToken(token)
	if err != nil {
		return nil, "", nil, err
	}

	members, total, rateLimitData, err := b.client.GetOrganizationMembers(
		ctx,
		resource.Id.Resource,
		limit,
		page,
	)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outputAnnotations, err
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	if len(members) == 0 {
		return nil, "", outputAnnotations, nil
	}

	var grants []*v2.Grant
	for _, member := range members {
		principalId, err := resourceSdk.NewResourceID(userResourceType, member.UserId)
		if err != nil {
			return nil, "", outputAnnotations, err
		}
		nextGrant := sdkGrant.NewGrant(
			resource,
			organizationEntitlementName,
			principalId,
		)
		grants = append(grants, nextGrant)
	}

	nextToken := client2.GetNextToken(page, limit, total)
	return grants, nextToken, outputAnnotations, nil
}

func (b *organizationBuilder) Grant(
	ctx context.Context,
	principal *v2.Resource,
	entitlement *v2.Entitlement,
) (
	annotations.Annotations,
	error,
) {
	l := ctxzap.Extract(ctx)
	userId := principal.Id.Resource
	organizationId := entitlement.Resource.Id.Resource
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"baton-auth0: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-auth0: only users can be granted organization membership")
	}

	var outputAnnotations annotations.Annotations
	rateLimitData, err := b.client.AddUserToOrganization(ctx, organizationId, userId)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return outputAnnotations, fmt.Errorf("baton-aouth0: failed to add user to organization: %s", err.Error())
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	return outputAnnotations, nil
}

func (b *organizationBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	entitlement := grant.Entitlement
	principal := grant.Principal
	organizationId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource

	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"baton-auth0: only users can have organization membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", userId),
		)
		return nil, fmt.Errorf("baton-auth0: only users can have organization membership revoked")
	}

	var outputAnnotations annotations.Annotations
	rateLimitData, err := b.client.RemoveUserFromOrganization(ctx, organizationId, userId)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return outputAnnotations, fmt.Errorf("baton-auth0: failed to revoke membership to organization: %w", err)
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	return outputAnnotations, nil
}

func newOrganizationBuilder(client *client2.Client) *organizationBuilder {
	return &organizationBuilder{client: client}
}
