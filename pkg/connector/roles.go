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

const roleEntitlementName = "assigned"

type roleBuilder struct {
	client *client.Client
}

func (o *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleResourceType
}

// Create a new connector resource for an Auth0 role.
func roleResource(role client.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	return resourceSdk.NewRoleResource(
		role.Name,
		roleResourceType,
		role.ID,
		[]resourceSdk.RoleTraitOption{
			resourceSdk.WithRoleProfile(
				map[string]interface{}{
					"id":          role.ID,
					"name":        role.Name,
					"description": role.Description,
				},
			),
		},
		resourceSdk.WithParentResourceID(parentResourceID),
	)
}

// List returns all the roles from the database as resource objects.
// Roles include a RoleTrait because they are the 'shape' of a standard role.
func (o *roleBuilder) List(
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
	logger.Debug("Starting Roles List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	page, limit, _, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	roles, total, ratelimitData, err := o.client.GetRoles(ctx, limit, page)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	if len(roles) == 0 {
		return outputResources, "", outputAnnotations, nil
	}

	for _, role := range roles {
		roleResource0, err := roleResource(role, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, roleResource0)
	}

	nextToken := client.GetNextToken(page, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for roles.
func (o *roleBuilder) Entitlements(
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
			roleEntitlementName,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(
				fmt.Sprintf("%s %s", resource.DisplayName, roleEntitlementName),
			),
			entitlement.WithDescription(
				fmt.Sprintf("Assigned %s role in Auth0", resource.DisplayName),
			),
		),
	}, "", nil, nil
}

// Grants always returns an empty slice for roles since they don't have any entitlements.
func (o *roleBuilder) Grants(
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
	page, limit, _, err := client.ParsePaginationToken(token)
	if err != nil {
		return nil, "", nil, err
	}

	users, total, ratelimitData, err := o.client.GetRoleUsers(
		ctx,
		resource.Id.Resource,
		limit,
		page,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	if len(users) == 0 {
		return nil, "", outputAnnotations, nil
	}

	var grants []*v2.Grant
	for _, user := range users {
		principalId, err := resourceSdk.NewResourceID(userResourceType, user.UserId)
		if err != nil {
			return nil, "", outputAnnotations, err
		}
		nextGrant := grant.NewGrant(
			resource,
			roleEntitlementName,
			principalId,
		)
		grants = append(grants, nextGrant)
	}

	nextToken := client.GetNextToken(page, limit, total)

	return grants, nextToken, outputAnnotations, nil
}

func (r *roleBuilder) Grant(
	ctx context.Context,
	principal *v2.Resource,
	entitlement *v2.Entitlement,
) (
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)
	userId := principal.Id.Resource
	roleId := entitlement.Resource.Id.Resource
	if principal.Id.ResourceType != userResourceType.Id {
		logger.Warn(
			"baton-auth0: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-auth0: only users can be granted role membership")
	}

	var outputAnnotations annotations.Annotations
	ratelimitData, err := r.client.AddUserToRole(ctx, roleId, userId)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-aouth0: failed to add user to role: %s", err.Error())
	}

	return outputAnnotations, nil
}

func (r *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	logger := ctxzap.Extract(ctx)
	entitlement := grant.Entitlement
	principal := grant.Principal
	roleId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource

	if principal.Id.ResourceType != userResourceType.Id {
		logger.Warn(
			"baton-auth0: only users can have role membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", userId),
		)
		return nil, fmt.Errorf("baton-auth0: only users can have role membership revoked")
	}

	var outputAnnotations annotations.Annotations
	ratelimitData, err := r.client.RemoveUserFromRole(ctx, roleId, userId)
	outputAnnotations.WithRateLimiting(ratelimitData)

	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-auth0: failed to revoke membership to role: %s", err.Error())
	}
	return outputAnnotations, nil
}

func newRoleBuilder(client *client.Client) *roleBuilder {
	return &roleBuilder{client: client}
}
