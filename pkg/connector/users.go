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

const workType = "WORK"
const workersSessionPrefix = "workers"
const workLocationsSessionPrefix = "work_locations"

// Page token prefixes to distinguish the phases of user List():
// Phase 1 (optional): pages through work locations, storing each in the session store.
// Phase 2: pages through workers, storing each page in the session store.
// Phase 3: pages through users, looking up cached workers and locations per-user.
const (
	locationsPagePrefix = "l:"
	workersPagePrefix   = "w:"
	usersPagePrefix     = "u:"
)

type userBuilder struct {
	client              *client.Client
	expandWorkLocations bool
	customFieldNames    []string
}

func (o *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return userResourceType
}

func userResource(user client.User, worker *client.Worker, workLocation *client.WorkLocation, customFieldNames []string) (*v2.Resource, error) {
	profile := map[string]any{
		"username": user.Username,
		"active":   user.Active,
		"locale":   user.Locale,
	}

	// Name fields from User
	if user.Name.GivenName != "" {
		profile["given_name"] = user.Name.GivenName
	}
	if user.Name.FamilyName != "" {
		profile["family_name"] = user.Name.FamilyName
	}
	if user.Name.MiddleName != "" {
		profile["middle_name"] = user.Name.MiddleName
	}
	if user.Name.Formatted != "" {
		profile["formatted_name"] = user.Name.Formatted
	}
	if user.Name.PreferredGivenName != "" {
		profile["preferred_given_name"] = user.Name.PreferredGivenName
	}
	if user.Name.PreferredFamilyName != "" {
		profile["preferred_family_name"] = user.Name.PreferredFamilyName
	}

	// Work address from User (first WORK-type address wins)
	for _, addr := range user.Addresses {
		if !strings.EqualFold(addr.Type, workType) {
			continue
		}
		addrMap := map[string]any{}
		if addr.Locality != "" {
			addrMap["locality"] = addr.Locality
			profile["locality"] = addr.Locality
		}
		if addr.Region != "" {
			addrMap["region"] = addr.Region
			profile["region"] = addr.Region
		}
		if addr.Country != "" {
			addrMap["country"] = addr.Country
			profile["country"] = addr.Country
		}
		if addr.StreetAddress != "" {
			addrMap["street_address"] = addr.StreetAddress
		}
		if addr.PostalCode != "" {
			addrMap["postal_code"] = addr.PostalCode
		}
		if len(addrMap) > 0 {
			profile["address"] = addrMap
		}
		break
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

		// Employment type information
		if worker.EmploymentType != nil {
			if worker.EmploymentType.Type != "" {
				profile["employment_type"] = worker.EmploymentType.Type
			}
			if worker.EmploymentType.Name != "" {
				profile["employment_name"] = worker.EmploymentType.Name
			}
			if worker.EmploymentType.Label != "" {
				profile["employment_label"] = worker.EmploymentType.Label
			}
		}

		// Department information
		if worker.DepartmentID != "" {
			profile["department_id"] = worker.DepartmentID
		}
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
		if worker.Manager != nil && worker.Manager.WorkEmail != "" {
			profile["manager_email"] = worker.Manager.WorkEmail
		}

		// Location information
		if worker.Location != nil && worker.Location.WorkLocationID != "" {
			profile["work_location_id"] = worker.Location.WorkLocationID
		}

		// Custom fields — only include fields whose names match the configured list.
		if len(customFieldNames) > 0 {
			allowed := make(map[string]bool, len(customFieldNames))
			for _, name := range customFieldNames {
				allowed[strings.ToLower(name)] = true
			}
			for _, cf := range worker.CustomFields {
				name := strings.TrimSpace(cf.Name)
				if name == "" || cf.Value == nil {
					continue
				}
				if !allowed[strings.ToLower(name)] {
					continue
				}
				key := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
				strVal := fmt.Sprintf("%v", cf.Value)
				if strVal == "" {
					continue
				}
				profile[key] = strVal
			}
		}
	}

	// Work location details (from separate API call)
	if workLocation != nil {
		if workLocation.Name != "" {
			profile["work_location_name"] = workLocation.Name
		}
		if workLocation.Address != nil {
			addrMap := map[string]any{}
			if workLocation.Address.Locality != "" {
				addrMap["locality"] = workLocation.Address.Locality
				// Work location address takes precedence over user WORK address
				// for top-level locality/region/country fields.
				profile["locality"] = workLocation.Address.Locality
			}
			if workLocation.Address.Region != "" {
				addrMap["region"] = workLocation.Address.Region
				// Work location address takes precedence over user WORK address
				// for top-level locality/region/country fields.
				profile["region"] = workLocation.Address.Region
			}
			if workLocation.Address.Country != "" {
				addrMap["country"] = workLocation.Address.Country
				// Work location address takes precedence over user WORK address
				// for top-level locality/region/country fields.
				profile["country"] = workLocation.Address.Country
			}
			if workLocation.Address.StreetAddress != "" {
				addrMap["street_address"] = workLocation.Address.StreetAddress
			}
			if workLocation.Address.PostalCode != "" {
				addrMap["postal_code"] = workLocation.Address.PostalCode
			}
			if len(addrMap) > 0 {
				profile["work_location_address"] = addrMap
			}
		}
	}

	// convert to time.Time
	createdAt, err := time.Parse(time.RFC3339, user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("baton-rippling: failed to parse created_at for user %s: %w", user.ID, err)
	}

	// Derive user status from worker status.
	var userStatus v2.UserTrait_Status_Status
	if worker != nil {
		switch worker.Status {
		case "ACTIVE":
			userStatus = v2.UserTrait_Status_STATUS_ENABLED
		case "TERMINATED":
			userStatus = v2.UserTrait_Status_STATUS_DISABLED
		}
	}

	userOpts := []resource.UserTraitOption{
		resource.WithUserProfile(profile),
		resource.WithCreatedAt(createdAt),
		resource.WithStatus(userStatus),
		resource.WithUserLogin(user.Username),
	}

	email := getWorkEmail(user.Emails)
	if email != nil {
		userOpts = append(userOpts, resource.WithEmail(email.Value, strings.EqualFold(email.Type, workType)))
	}

	return resource.NewUserResource(
		user.DisplayName,
		userResourceType,
		user.ID,
		userOpts,
	)
}

// listWorkLocationPage fetches one page of work locations from the API and
// stores them in the session store. Returns no resources — this phase only populates the cache.
func (o *userBuilder) listWorkLocationPage(ctx context.Context, pageToken string, ss sessions.SessionStore) ([]*v2.Resource, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	resp, ratelimitData, err := o.client.ListWorkLocations(ctx, pageToken)
	annos = *annos.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list work locations: %w", err)
	}

	toStore := make(map[string]client.WorkLocation, len(resp.Results))
	for _, loc := range resp.Results {
		if loc.ID != "" {
			toStore[loc.ID] = loc
		}
	}
	if len(toStore) > 0 {
		if err := session.SetManyJSON(ctx, ss, toStore, sessions.WithPrefix(workLocationsSessionPrefix)); err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to store work locations in session store: %w", err)
		}
	}

	// If more pages remain, stay in the locations phase.
	// Otherwise transition to the workers phase.
	nextToken := workersPagePrefix
	if resp.NextLink != "" {
		nextToken = locationsPagePrefix + resp.NextLink
	}

	return nil, &resource.SyncOpResults{
		NextPageToken: nextToken,
		Annotations:   annos,
	}, nil
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

	userIDs := make([]string, 0, len(usersResponse.Results))
	for _, user := range usersResponse.Results {
		userIDs = append(userIDs, user.ID)
	}

	workers, err := session.GetManyJSON[client.Worker](ctx, ss, userIDs, sessions.WithPrefix(workersSessionPrefix))
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to get workers from session store: %w", err)
	}

	// Batch-fetch work locations for all workers on this page.
	var workLocations map[string]client.WorkLocation
	if o.expandWorkLocations {
		locationIDs := make([]string, 0)
		seen := make(map[string]bool)
		for _, user := range usersResponse.Results {
			if worker, found := workers[user.ID]; found {
				if worker.Location != nil && worker.Location.WorkLocationID != "" && !seen[worker.Location.WorkLocationID] {
					locationIDs = append(locationIDs, worker.Location.WorkLocationID)
					seen[worker.Location.WorkLocationID] = true
				}
			}
		}
		if len(locationIDs) > 0 {
			workLocations, err = session.GetManyJSON[client.WorkLocation](ctx, ss, locationIDs, sessions.WithPrefix(workLocationsSessionPrefix))
			if err != nil {
				return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to get work locations from session store: %w", err)
			}
		}
	}

	rv := make([]*v2.Resource, 0, len(usersResponse.Results))
	for _, user := range usersResponse.Results {
		var workerPtr *client.Worker
		if worker, found := workers[user.ID]; found {
			workerPtr = &worker
		}

		var workLocationPtr *client.WorkLocation
		if o.expandWorkLocations && workerPtr != nil && workerPtr.Location != nil && workerPtr.Location.WorkLocationID != "" {
			if wl, ok := workLocations[workerPtr.Location.WorkLocationID]; ok {
				workLocationPtr = &wl
			}
		}

		r, err := userResource(user, workerPtr, workLocationPtr, o.customFieldNames)
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
	case o.expandWorkLocations && (token == "" || strings.HasPrefix(token, locationsPagePrefix)):
		// Phase 1 (optional): page through work locations, storing each in the session store.
		return o.listWorkLocationPage(ctx, strings.TrimPrefix(token, locationsPagePrefix), opts.Session)

	case token == "" || strings.HasPrefix(token, workersPagePrefix):
		// Phase 2: page through workers, storing each page in the session store.
		return o.listWorkerPage(ctx, strings.TrimPrefix(token, workersPagePrefix), opts.Session)

	default:
		// Phase 3: page through users, looking up cached workers and locations per-user.
		return o.listUserPage(ctx, strings.TrimPrefix(token, usersPagePrefix), opts.Session)
	}
}

func getWorkEmail(emails []client.Email) *client.Email {
	var workEmail *client.Email
	for _, e := range emails {
		if strings.EqualFold(e.Type, workType) {
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

func newUserBuilder(client *client.Client, expandWorkLocations bool, customFieldNames []string) *userBuilder {
	return &userBuilder{
		client:              client,
		expandWorkLocations: expandWorkLocations,
		customFieldNames:    customFieldNames,
	}
}
