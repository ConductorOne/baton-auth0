package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const organizationEntitlementName = "member"

type organizationBuilder struct {
	client *client.Client
}

func (o *organizationBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return organizationResourceType
}

// Create a new connector resource for an Auth0 organization.
func organizationResource(
	organization client.Organization,
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
func (o *organizationBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	pToken *pagination.Token,
) (
	[]*v2.Resource,
	string,
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Starting Organizations List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	offset, limit, _, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	organizations, total, ratelimitData, err := o.client.GetOrganizations(ctx, limit, offset)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	for _, organization := range organizations {
		organizationResource0, err := organizationResource(organization, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, organizationResource0)
	}

	nextToken := client.GetNextToken(offset, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

func (o *organizationBuilder) Entitlements(
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
		entitlement.NewAssignmentEntitlement(
			resource,
			organizationEntitlementName,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(
				fmt.Sprintf("%s %s", resource.DisplayName, organizationEntitlementName),
			),
			entitlement.WithDescription(
				fmt.Sprintf("Member of %s organization in Auth0", resource.DisplayName),
			),
		),
	}, "", nil, nil
}

func (o *organizationBuilder) Grants(
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
	offset, limit, _, err := client.ParsePaginationToken(token)
	if err != nil {
		return nil, "", nil, err
	}

	members, total, ratelimitData, err := o.client.GetOrganizationMembers(
		ctx,
		resource.Id.Resource,
		limit,
		offset,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	var grants []*v2.Grant
	for _, member := range members {
		principalId, err := resourceSdk.NewResourceID(userResourceType, member.UserId)
		if err != nil {
			return nil, "", outputAnnotations, err
		}
		nextGrant := grant.NewGrant(
			resource,
			organizationEntitlementName,
			principalId,
		)
		grants = append(grants, nextGrant)
	}

	nextToken := client.GetNextToken(offset, limit, total)

	return grants, nextToken, outputAnnotations, nil
}

func (o *organizationBuilder) Grant(
	ctx context.Context,
	principal *v2.Resource,
	entitlement *v2.Entitlement,
) (
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)
	userId := principal.Id.Resource
	organizationId := entitlement.Resource.Id.Resource
	if principal.Id.ResourceType != userResourceType.Id {
		logger.Warn(
			"baton-auth0: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-auth0: only users can be granted organization membership")
	}

	var outputAnnotations annotations.Annotations
	ratelimitData, err := o.client.AddUserToOrganization(ctx, organizationId, userId)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-aouth0: failed to add user to organization: %s", err.Error())
	}

	return outputAnnotations, nil
}

func (o *organizationBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	logger := ctxzap.Extract(ctx)
	entitlement := grant.Entitlement
	principal := grant.Principal
	organizationId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource

	if principal.Id.ResourceType != userResourceType.Id {
		logger.Warn(
			"baton-auth0: only users can have organization membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", userId),
		)
		return nil, fmt.Errorf("baton-auth0: only users can have organization membership revoked")
	}

	var outputAnnotations annotations.Annotations
	ratelimitData, err := o.client.RemoveUserFromOrganization(ctx, organizationId, userId)
	outputAnnotations.WithRateLimiting(ratelimitData)

	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-auth0: failed to revoke membership to organization: %s", err.Error())
	}
	return outputAnnotations, nil
}

func newOrganizationBuilder(client *client.Client) *organizationBuilder {
	return &organizationBuilder{client: client}
}
