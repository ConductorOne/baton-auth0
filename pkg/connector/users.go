package connector

import (
	"context"
	"time"

	"github.com/conductorone/baton-auth0/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type userBuilder struct {
	client              *client.Client
	syncUsersByJob      bool
	syncUsersByJobLimit int
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
		resourceSdk.WithCreatedAt(user.CreatedAt),
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

	if o.syncUsersByJob {
		type syncJobPage struct {
			Id      string
			Attempt int
		}

		bag, err := pagination.GenBagFromToken[syncJobPage](pToken)
		if err != nil {
			return nil, "", nil, err
		}

		if bag.Current() == nil {
			job, ratelimitData, err := o.client.CreateJob(ctx, o.syncUsersByJobLimit)
			outputAnnotations.WithRateLimiting(ratelimitData)
			if err != nil {
				return nil, "", outputAnnotations, err
			}

			bag.Push(syncJobPage{
				Id:      job.Id,
				Attempt: 0,
			})

			nextToken, err := bag.Marshal()
			if err != nil {
				return nil, "", outputAnnotations, err
			}

			return nil, nextToken, outputAnnotations, nil
		}

		state := bag.Pop()

		job, ratelimitData, err := o.client.GetJob(ctx, state.Id)
		outputAnnotations.WithRateLimiting(ratelimitData)
		if err != nil {
			return nil, "", outputAnnotations, err
		}

		logger.Debug("Sync job status", zap.String("job_id", job.Id), zap.String("status", job.Status))

		if job.Status != "completed" {
			var anno annotations.Annotations

			anno.WithRateLimiting(&v2.RateLimitDescription{
				Limit:     1,
				Remaining: 0,
				ResetAt:   timestamppb.New(time.Now().Add(time.Second * 10)),
			})
			bag.Push(syncJobPage{
				Id:      state.Id,
				Attempt: state.Attempt + 1,
			})

			nextToken, err := bag.Marshal()
			if err != nil {
				return nil, "", anno, err
			}

			return nil, nextToken, anno, status.Errorf(codes.Unavailable, "Sync job it's not completed: %s", job.Status)
		}

		logger.Debug("Sync job completed", zap.String("job_id", state.Id))

		usersJob, err := o.client.ProcessUserJob(ctx, job)
		if err != nil {
			return nil, "", nil, err
		}

		for _, user := range usersJob {
			userResource0, err := userResource(user, parentResourceID)
			if err != nil {
				return nil, "", nil, err
			}
			outputResources = append(outputResources, userResource0)
		}

		return outputResources, "", nil, nil
	}

	page, limit, _, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	users, total, ratelimitData, err := o.client.GetUsers(ctx, limit, page)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}

	if len(users) == 0 {
		return outputResources, "", outputAnnotations, nil
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
	nextToken := client.GetNextToken(page, limit, total)

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

func newUserBuilder(
	client *client.Client,
	syncUsersByJob bool,
	syncUsersByJobLimit int,
) *userBuilder {
	return &userBuilder{
		client:              client,
		syncUsersByJob:      syncUsersByJob,
		syncUsersByJobLimit: syncUsersByJobLimit,
	}
}
