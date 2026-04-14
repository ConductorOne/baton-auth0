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
	_ connectorbuilder.ResourceSyncer      = (*roleBuilder)(nil)
	_ connectorbuilder.ResourceProvisioner = (*roleBuilder)(nil)
)

const roleEntitlementName = "assigned"
const rolePermissionEntitlementName = "has_permission"

type roleBuilder struct {
	client          *client2.Client
	syncPermissions bool
}

func (b *roleBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return roleResourceType
}

// Create a new connector resource for an Auth0 role.
func roleResource(role client2.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
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
func (b *roleBuilder) List(
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

	roles, total, rateLimitData, err := b.client.GetRoles(ctx, limit, page)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outputAnnotations, err
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

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

	nextToken := client2.GetNextToken(page, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

func (b *roleBuilder) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	ents := []*v2.Entitlement{
		sdkEntitlement.NewAssignmentEntitlement(
			resource,
			roleEntitlementName,
			sdkEntitlement.WithGrantableTo(userResourceType),
			sdkEntitlement.WithDisplayName(
				fmt.Sprintf("%s %s", resource.DisplayName, roleEntitlementName),
			),
			sdkEntitlement.WithDescription(
				fmt.Sprintf("Assigned %s role in Auth0", resource.DisplayName),
			),
		),
	}

	if b.syncPermissions {
		ents = append(
			ents,
			sdkEntitlement.NewPermissionEntitlement(
				resource,
				rolePermissionEntitlementName,
				sdkEntitlement.WithGrantableTo(scopeResourceType),
				sdkEntitlement.WithDisplayName(
					fmt.Sprintf("%s %s", resource.DisplayName, rolePermissionEntitlementName),
				),
				sdkEntitlement.WithDescription(
					fmt.Sprintf("Has %s role permissions in Auth0", resource.DisplayName),
				),
				sdkEntitlement.WithAnnotation(&v2.EntitlementImmutable{}),
			),
		)
	}

	return ents, "", nil, nil
}

func (b *roleBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	token *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	var bag pagination.Bag

	err := bag.Unmarshal(token.Token)
	if err != nil {
		return nil, "", nil, err
	}
	if bag.Current() == nil {
		bag.Push(pagination.PageState{
			ResourceTypeID: userResourceType.Id,
		})

		if b.syncPermissions {
			bag.Push(pagination.PageState{
				ResourceTypeID: scopeResourceType.Id,
			})
		}

		nextToken, err := bag.Marshal()
		if err != nil {
			return nil, "", nil, err
		}

		return nil, nextToken, nil, nil
	}

	state := bag.Current()

	switch state.ResourceTypeID {
	case userResourceType.Id:
		var outputAnnotations annotations.Annotations

		// Auth0's page-based pagination for this endpoint has a hard 1000-record cap.
		// Checkpoint pagination ("from"/"take") has no such limit.
		from, err := client2.ParseRoleUserCheckpointToken(state.Token)
		if err != nil {
			return nil, "", nil, err
		}

		users, next, rateLimitData, err := b.client.GetRoleUsersCheckpoint(
			ctx,
			resource.Id.Resource,
			from,
			client2.PageSizeDefault,
		)
		if err != nil {
			if rateLimitData != nil {
				outputAnnotations.WithRateLimiting(rateLimitData)
			}
			return nil, "", outputAnnotations, err
		}
		outputAnnotations.WithRateLimiting(rateLimitData)

		if len(users) == 0 {
			return nil, "", outputAnnotations, nil
		}

		var grants []*v2.Grant
		for _, user := range users {
			principalId, err := resourceSdk.NewResourceID(userResourceType, user.UserId)
			if err != nil {
				return nil, "", outputAnnotations, err
			}
			nextGrant := sdkGrant.NewGrant(
				resource,
				roleEntitlementName,
				principalId,
			)
			grants = append(grants, nextGrant)
		}

		nextToken, err := bag.NextToken(client2.GetNextRoleUserCheckpointToken(next))
		if err != nil {
			return nil, "", nil, err
		}

		return grants, nextToken, outputAnnotations, nil
	case scopeResourceType.Id:
		var outputAnnotations annotations.Annotations

		permissions, rateLimitData, err := b.client.GetRolePermissions(
			ctx,
			resource.Id.Resource,
		)
		if err != nil {
			if rateLimitData != nil {
				outputAnnotations.WithRateLimiting(rateLimitData)
			}
			return nil, "", outputAnnotations, err
		}
		outputAnnotations.WithRateLimiting(rateLimitData)

		if len(permissions) == 0 {
			return nil, "", outputAnnotations, nil
		}

		var grants []*v2.Grant
		for _, permission := range permissions {
			// Same as formatScopeId function in scope.go
			scopeId := fmt.Sprintf("%s:%s", permission.ResourceServerIdentifier, permission.PermissionName)

			principalId, err := resourceSdk.NewResourceID(scopeResourceType, scopeId)
			if err != nil {
				return nil, "", outputAnnotations, err
			}
			nextGrant := sdkGrant.NewGrant(
				resource,
				rolePermissionEntitlementName,
				principalId,
			)
			grants = append(grants, nextGrant)
		}

		bag.Pop()
		nextToken, err := bag.Marshal()
		if err != nil {
			return nil, "", nil, err
		}

		return grants, nextToken, outputAnnotations, nil
	default:
		return nil, "", nil, fmt.Errorf("baton-auth0: unknown resource type %s", state.ResourceTypeID)
	}
}

func (b *roleBuilder) Grant(
	ctx context.Context,
	principal *v2.Resource,
	entitlement *v2.Entitlement,
) (
	annotations.Annotations,
	error,
) {
	l := ctxzap.Extract(ctx)
	userId := principal.Id.Resource
	roleId := entitlement.Resource.Id.Resource
	if principal.Id.ResourceType != userResourceType.Id {
		l.Debug(
			"baton-auth0: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-auth0: only users can be granted role membership")
	}

	var outputAnnotations annotations.Annotations
	rateLimitData, err := b.client.AddUserToRole(ctx, roleId, userId)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return outputAnnotations, fmt.Errorf("baton-auth0: failed to add user to role: %w", err)
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	return outputAnnotations, nil
}

func (b *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	entitlement := grant.Entitlement
	principal := grant.Principal
	roleId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource

	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"baton-auth0: only users can have role membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", userId),
		)
		return nil, fmt.Errorf("baton-auth0: only users can have role membership revoked")
	}

	var outputAnnotations annotations.Annotations
	rateLimitData, err := b.client.RemoveUserFromRole(ctx, roleId, userId)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return outputAnnotations, fmt.Errorf("baton-auth0: failed to revoke membership to role: %w", err)
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	return outputAnnotations, nil
}

func newRoleBuilder(client *client2.Client, syncPermissions bool) *roleBuilder {
	return &roleBuilder{
		client:          client,
		syncPermissions: syncPermissions,
	}
}
