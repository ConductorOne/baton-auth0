package connector

import (
	"context"
	"time"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

var _ connectorbuilder.ResourceSyncer = (*userBuilder)(nil)

type userBuilder struct {
	client *client.Client
}

func (b *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
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
func (b *userBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	pToken *pagination.Token,
) (
	[]*v2.Resource,
	string,
	annotations.Annotations,
	error,
) {
	l := ctxzap.Extract(ctx)

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	page, limit, since, until, newestUserCreationDate, err := client.ParseUserPaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	users, total, rateLimitData, err := b.client.GetUsers(ctx, limit, page, since, until)
	if err != nil {
		if rateLimitData != nil {
			outputAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outputAnnotations, err
	}
	outputAnnotations.WithRateLimiting(rateLimitData)

	if len(users) == 0 {
		return outputResources, "", outputAnnotations, nil
	}

	var newestCreatedAt time.Time
	if newestUserCreationDate != nil {
		newestCreatedAt = *newestUserCreationDate
	}

	for _, user := range users {
		if user.CreatedAt.UTC().After(newestCreatedAt) {
			newestCreatedAt = user.CreatedAt.UTC()
		}

		userResource0, err := userResource(user, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, userResource0)
	}

	// Auth0's User Search API enforces a hard cap of 1,000 results, even when paginating.
	// Requesting beyond this limit returns a 400 error.
	// See https://auth0.com/docs/manage-users/user-search/view-search-results-by-page#limitation.
	if total > client.Auth0UserSearchMaxResults {
		l.Debug(
			"Auth0 user search exceeds 1000-result API limit; using date-range windowing to fetch remaining users.",
			zap.Int("total_users", total),
			zap.Int("api_limit", client.Auth0UserSearchMaxResults),
		)
		total = client.Auth0UserSearchMaxResults
	}

	nextToken, err := client.GetNextUsersToken(page, limit, total, since, &newestCreatedAt)
	if err != nil {
		return nil, "", nil, err
	}

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
func (b *userBuilder) Entitlements(
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
func (b *userBuilder) Grants(
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
