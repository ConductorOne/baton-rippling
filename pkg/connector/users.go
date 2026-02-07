package connector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

const workEmailType = "WORK"

type userBuilder struct {
	client *client.Client
}

func (o *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return userResourceType
}

func userResource(user client.User, worker *client.Worker) (*v2.Resource, error) {
	profile := map[string]any{
		"username": user.Username,
		"active":   user.Active,
		"locale":   user.Locale,
	}

	// Add worker-related attributes if available
	if worker != nil {
		// Employment information
		if worker.Title != "" {
			profile["title"] = worker.Title
		}
		if worker.Status != "" {
			profile["status"] = worker.Status
		}
		if worker.StartDate != "" {
			profile["start_date"] = worker.StartDate
		}
		if worker.EndDate != "" {
			profile["end_date"] = worker.EndDate
		}
		if worker.WorkEmail != "" {
			profile["work_email"] = worker.WorkEmail
		}
		if worker.Country != "" {
			profile["country"] = worker.Country
		}

		// Employment type information
		if worker.EmploymentType != nil {
			employmentType := map[string]any{}
			if worker.EmploymentType.Label != "" {
				employmentType["label"] = worker.EmploymentType.Label
			}
			if worker.EmploymentType.Name != "" {
				employmentType["name"] = worker.EmploymentType.Name
			}
			if worker.EmploymentType.Type != "" {
				employmentType["type"] = worker.EmploymentType.Type
			}
			if len(employmentType) > 0 {
				profile["employment_type"] = employmentType
			}
		}

		// Department information
		if worker.Department != nil && worker.Department.Name != "" {
			profile["department"] = worker.Department.Name
		}

		// Level/rank information
		if worker.Level != nil && worker.Level.Name != "" {
			profile["level"] = worker.Level.Name
		}

		// Manager information
		if worker.IsManager {
			profile["is_manager"] = true
		}
		if worker.ManagerID != "" {
			profile["manager_id"] = worker.ManagerID
		}

		// Location information
		if worker.Location != nil && worker.Location.WorkLocationID != "" {
			profile["work_location_id"] = worker.Location.WorkLocationID
		}
	}

	// convert to time.Time
	createdAt, err := time.Parse(time.RFC3339, user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("baton-rippling: failed to parse created_at for user %s: %w", user.ID, err)
	}

	userOpts := []resource.UserTraitOption{
		resource.WithUserProfile(profile),
		resource.WithCreatedAt(createdAt),
	}

	email := getWorkEmail(user.Emails)
	if email != nil {
		userOpts = append(userOpts, resource.WithEmail(email.Value, strings.EqualFold(email.Type, workEmailType)))
	}

	return resource.NewUserResource(
		user.Name.DisplayName,
		userResourceType,
		user.ID,
		userOpts,
	)
}

func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	usersResponse, ratelimitData, err := o.client.ListUsers(ctx, pToken.Token)
	annotations = *annotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-rippling: failed to list users: %w", err)
	}

	// Fetch workers to get additional profile attributes
	workersResponse, workersRatelimitData, err := o.client.ListWorkers(ctx, "")
	if workersRatelimitData != nil {
		annotations = *annotations.WithRateLimiting(workersRatelimitData)
	}
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-rippling: failed to list workers: %w", err)
	}

	// Create a lookup map of workers by user ID for efficient access
	workersByUserID := make(map[string]*client.Worker)
	for i := range workersResponse.Results {
		worker := &workersResponse.Results[i]
		if worker.UserID != "" {
			workersByUserID[worker.UserID] = worker
		}
	}

	rv := []*v2.Resource{}
	for _, user := range usersResponse.Results {
		// Look up worker data for this user
		worker := workersByUserID[user.ID]
		resource, err := userResource(user, worker)
		if err != nil {
			return nil, "", annotations, fmt.Errorf("baton-rippling: failed to convert user %s to resource: %w", user.ID, err)
		}
		rv = append(rv, resource)
	}

	return rv, usersResponse.NextLink, annotations, nil
}

func getWorkEmail(emails []client.Email) *client.Email {
	var workEmail *client.Email
	for _, e := range emails {
		if strings.EqualFold(e.Type, workEmailType) {
			return &e
		}

		if workEmail == nil {
			workEmail = &e
		}
	}

	return workEmail
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
