package connector

import (
	"context"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type userBuilder struct {
	client *client.Client
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// Create a new connector resource for an Auth0 user.
func userResource(user client.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	firstName, lastName := resourceSdk.SplitFullName(user.Name)

	profile := map[string]interface{}{
		"id":         user.UserId,
		"email":      user.Email,
		"first_name": firstName,
		"last_name":  lastName,
		"nickname":   user.Nickname,
	}

	userTraitOptions := []resourceSdk.UserTraitOption{
		resourceSdk.WithEmail(user.Email, true),
		resourceSdk.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
		resourceSdk.WithUserProfile(profile),
		resourceSdk.WithUserLogin(user.Email),
	}

	return resourceSdk.NewUserResource(
		user.Nickname,
		userResourceType,
		user.UserId,
		userTraitOptions,
		resourceSdk.WithParentResourceID(parentResourceID),
	)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(
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
	logger.Debug("Starting Users List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	offset, limit, _, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	users, total, ratelimitData, err := o.client.GetUsers(ctx, limit, offset)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	for _, user := range users {
		userResource0, err := userResource(user, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, userResource0)
	}

	// TODO(marcos): it might never be possible to get a second page if we are limited to 1,000 results.
	// See https://auth0.com/docs/users/search/v3/view-search-results-by-page#limitation.
	nextToken := client.GetNextToken(offset, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(
	_ context.Context,
	_ *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(
	_ context.Context,
	_ *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{client: client}
}
