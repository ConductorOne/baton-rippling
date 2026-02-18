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
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/types/sessions"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const workEmailType = "WORK"
const workersSessionPrefix = "workers"

// Page token prefixes to distinguish the two phases of user List():
// Phase 1 pages through workers, storing each page in the session store.
// Phase 2 pages through users, looking up cached workers per-user.
const (
	workersPagePrefix = "w:"
	usersPagePrefix   = "u:"
)

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

// listWorkerPage fetches one page of workers from the API and stores them in
// the session store. Returns no resources — this phase only populates the cache.
func (o *userBuilder) listWorkerPage(ctx context.Context, pageToken string, ss sessions.SessionStore) ([]*v2.Resource, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	workersResponse, ratelimitData, err := o.client.ListWorkers(ctx, pageToken)
	annos = *annos.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list workers: %w", err)
	}

	toStore := make(map[string]client.Worker, len(workersResponse.Results))
	for _, worker := range workersResponse.Results {
		if worker.UserID != "" {
			toStore[worker.UserID] = worker
		}
	}
	if len(toStore) > 0 {
		if err := session.SetManyJSON(ctx, ss, toStore, sessions.WithPrefix(workersSessionPrefix)); err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to store workers in session store: %w", err)
		}
	}

	// If more worker pages remain, stay in the workers phase.
	// Otherwise transition to the users phase.
	nextToken := usersPagePrefix
	if workersResponse.NextLink != "" {
		nextToken = workersPagePrefix + workersResponse.NextLink
	}

	return nil, &resource.SyncOpResults{
		NextPageToken: nextToken,
		Annotations:   annos,
	}, nil
}

// listUserPage fetches one page of users, enriching each with its cached worker data.
func (o *userBuilder) listUserPage(ctx context.Context, pageToken string, ss sessions.SessionStore) ([]*v2.Resource, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	usersResponse, ratelimitData, err := o.client.ListUsers(ctx, pageToken)
	annos = *annos.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list users: %w", err)
	}

	rv := make([]*v2.Resource, 0, len(usersResponse.Results))
	for _, user := range usersResponse.Results {
		worker, found, err := session.GetJSON[client.Worker](ctx, ss, user.ID, sessions.WithPrefix(workersSessionPrefix))
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to get worker for user %s from session store: %w", user.ID, err)
		}

		var workerPtr *client.Worker
		if found {
			workerPtr = &worker
		}

		r, err := userResource(user, workerPtr)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to convert user %s to resource: %w", user.ID, err)
		}
		rv = append(rv, r)
	}

	nextToken := ""
	if usersResponse.NextLink != "" {
		nextToken = usersPagePrefix + usersResponse.NextLink
	}

	return rv, &resource.SyncOpResults{
		NextPageToken: nextToken,
		Annotations:   annos,
	}, nil
}

func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, opts resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	token := opts.PageToken.Token

	switch {
	case token == "" || strings.HasPrefix(token, workersPagePrefix):
		// Phase 1: page through workers, storing each page in the session store.
		return o.listWorkerPage(ctx, strings.TrimPrefix(token, workersPagePrefix), opts.Session)

	default:
		// Phase 2: page through users, looking up cached workers per-user.
		return o.listUserPage(ctx, strings.TrimPrefix(token, usersPagePrefix), opts.Session)
	}
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

// Grants returns team membership grants for this user by looking up the user's
// worker record from the session store and creating a grant for each team.
func (o *userBuilder) Grants(ctx context.Context, r *v2.Resource, opts resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	worker, found, err := session.GetJSON[client.Worker](ctx, opts.Session, r.Id.Resource, sessions.WithPrefix(workersSessionPrefix))
	if err != nil {
		return nil, nil, fmt.Errorf("baton-rippling: failed to get worker for user %s from session store: %w", r.Id.Resource, err)
	}
	if !found {
		l := ctxzap.Extract(ctx)
		l.Debug("no worker found for user, skipping team grants", zap.String("user_id", r.Id.Resource))
		return nil, nil, nil
	}
	if worker.Status == "TERMINATED" {
		l := ctxzap.Extract(ctx)
		l.Debug("worker is terminated, skipping team grants", zap.String("user_id", r.Id.Resource))
		return nil, nil, nil
	}

	rv := make([]*v2.Grant, 0, len(worker.TeamsID))
	for _, teamID := range worker.TeamsID {
		teamResourceID, err := resource.NewResourceID(teamResourceType, teamID)
		if err != nil {
			return nil, nil, fmt.Errorf("baton-rippling: failed to create resource ID for team %s: %w", teamID, err)
		}
		teamResource := &v2.Resource{Id: teamResourceID}
		rv = append(rv, grant.NewGrant(
			teamResource,
			teamMembership,
			r.Id,
		))
	}

	return rv, nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
