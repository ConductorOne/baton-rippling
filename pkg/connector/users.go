package connector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/session"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/types/sessions"
)

const workEmailType = "WORK"
const workersSessionPrefix = "workers"

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
		profile["is_manager"] = worker.IsManager
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

// fetchWorkers fetches all workers from the API and stores them in the session store,
// keyed by user ID. On subsequent calls within the same sync, workers are read from
// the session store instead of re-fetching from the API.
func (o *userBuilder) fetchWorkers(ctx context.Context, ss sessions.SessionStore, annos *annotations.Annotations) (map[string]*client.Worker, error) {
	cache := session.NewJSONSessionCache[client.Worker](ss)

	// Check if workers have already been cached in the session store.
	existing, _, err := cache.GetAll(ctx, "", sessions.WithPrefix(workersSessionPrefix))
	if err != nil {
		return nil, fmt.Errorf("baton-rippling: failed to read workers from session store: %w", err)
	}
	if len(existing) > 0 {
		result := make(map[string]*client.Worker, len(existing))
		for userID, worker := range existing {
			w := worker
			result[userID] = &w
		}
		return result, nil
	}

	// Fetch all pages of workers from the API.
	toStore := make(map[string]client.Worker)
	nextLink := ""
	for {
		workersResponse, workersRatelimitData, workersErr := o.client.ListWorkers(ctx, nextLink)
		if workersRatelimitData != nil {
			*annos = *annos.WithRateLimiting(workersRatelimitData)
		}
		if workersErr != nil {
			return nil, fmt.Errorf("baton-rippling: failed to list workers: %w", workersErr)
		}
		for _, worker := range workersResponse.Results {
			if worker.UserID != "" {
				toStore[worker.UserID] = worker
			}
		}
		if workersResponse.NextLink == "" {
			break
		}
		nextLink = workersResponse.NextLink
	}

	// Persist to session store for subsequent pages.
	if len(toStore) > 0 {
		if err := cache.SetMany(ctx, toStore, sessions.WithPrefix(workersSessionPrefix)); err != nil {
			return nil, fmt.Errorf("baton-rippling: failed to store workers in session store: %w", err)
		}
	}

	result := make(map[string]*client.Worker, len(toStore))
	for userID, worker := range toStore {
		w := worker
		result[userID] = &w
	}
	return result, nil
}

func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, opts resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	usersResponse, ratelimitData, err := o.client.ListUsers(ctx, opts.PageToken.Token)
	annos = *annos.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list users: %w", err)
	}

	// Fetch workers from session store (or API on first call).
	workersByUserID, err := o.fetchWorkers(ctx, opts.Session, &annos)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, err
	}

	rv := make([]*v2.Resource, 0, len(usersResponse.Results))
	for _, user := range usersResponse.Results {
		worker := workersByUserID[user.ID]
		r, err := userResource(user, worker)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to convert user %s to resource: %w", user.ID, err)
		}
		rv = append(rv, r)
	}

	return rv, &resource.SyncOpResults{
		NextPageToken: usersResponse.NextLink,
		Annotations:   annos,
	}, nil
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
func (o *userBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(_ context.Context, _ *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	return nil, nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
