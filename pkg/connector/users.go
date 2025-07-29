package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	client *client.Client
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

func userResource(user client.User) (*v2.Resource, error) {
	profile := map[string]any{
		"username": user.Username,
		"active":   user.Active,
		"locale":   user.Locale,
	}

	// convert to time.Time
	createdAt, err := time.Parse(time.RFC3339, user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("baton-rippling: failed to parse created_at for user %s: %w", user.ID, err)
	}

	email := ""
	if len(user.Emails) > 0 {
		email = user.Emails[0].Value
	}

	return resource.NewUserResource(
		user.Name.DisplayName,
		userResourceType,
		user.ID,
		[]resource.UserTraitOption{
			resource.WithUserProfile(profile),
			resource.WithCreatedAt(createdAt),
			resource.WithEmail(email, true),
		},
	)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {

	var annotations annotations.Annotations
	usersResponse, ratelimitData, err := o.client.ListUsers(ctx, pToken.Token)
	annotations = *annotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-rippling: failed to list users: %w", err)
	}

	rv := []*v2.Resource{}
	for _, user := range usersResponse.Results {
		resource, err := userResource(user)
		if err != nil {
			return nil, "", annotations, fmt.Errorf("baton-rippling: failed to convert user %s to resource: %w", user.ID, err)
		}
		rv = append(rv, resource)
	}

	return rv, usersResponse.NextLink, annotations, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
