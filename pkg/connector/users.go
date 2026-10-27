package connector

import (
	"context"
	"encoding/json"
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
	"google.golang.org/protobuf/proto"
)

const workType = "WORK"
const workersSessionPrefix = "workers"
const workLocationsSessionPrefix = "work_locations"

type userBuilder struct {
	client              *client.Client
	resourceType        *v2.ResourceType
	expandWorkLocations bool
	customFieldNames    []string
}

func (o *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
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
				if _, exists := profile[key]; exists {
					continue // don't overwrite built-in profile fields
				}
				var strVal string
				switch v := cf.Value.(type) {
				case string:
					strVal = v
				default:
					b, _ := json.Marshal(v)
					strVal = string(b)
				}
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

	// Rippling's user.Username is not guaranteed to be the work email — it can
	// be a personal/pre-boarding address. Emit the work email as a login alias
	// so SSO lookups and cross-connector AppUser matching resolve to this user.
	var aliases []string
	if workEmail := resolveWorkEmail(user, worker); workEmail != "" && !strings.EqualFold(workEmail, user.Username) {
		aliases = append(aliases, workEmail)
	}

	userOpts := []resource.UserTraitOption{
		resource.WithUserLogin(user.Username, aliases...),
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
		resource.WithResourceProfile(profile),
		resource.WithResourceCreatedAt(createdAt),
		resource.WithResourceStatus(v2.Status_ResourceStatus(userStatus), ""),
	)
}

func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, opts resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	workersResponse, ratelimitData, err := o.client.ListWorkers(ctx, opts.PageToken.Token)
	annos = *annos.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list workers: %w", err)
	}

	rv := make([]*v2.Resource, 0, len(workersResponse.Results))
	workersToStore := make(map[string]client.Worker, len(workersResponse.Results))
	locationCache := make(map[string]*client.WorkLocation)

	for _, worker := range workersResponse.Results {
		if worker.User == nil || worker.User.ID == "" {
			continue
		}
		user := *worker.User

		workLocationPtr, rl, err := o.resolveWorkLocation(ctx, opts.Session, worker.Location, locationCache)
		annos = *annos.WithRateLimiting(rl)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, err
		}

		r, err := userResource(user, &worker, workLocationPtr, o.customFieldNames)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to convert user %s to resource: %w", user.ID, err)
		}
		rv = append(rv, r)
		workersToStore[user.ID] = worker
	}

	if len(workersToStore) > 0 {
		if err := session.SetManyJSON(ctx, opts.Session, workersToStore, sessions.WithPrefix(workersSessionPrefix)); err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to store workers in session store: %w", err)
		}
	}

	return rv, &resource.SyncOpResults{
		NextPageToken: workersResponse.NextLink,
		Annotations:   annos,
	}, nil
}

// Worker locations are critical HR data.
func (o *userBuilder) resolveWorkLocation(ctx context.Context, ss sessions.SessionStore, loc *client.Location, perPage map[string]*client.WorkLocation) (*client.WorkLocation, *v2.RateLimitDescription, error) {
	if !o.expandWorkLocations || loc == nil || loc.WorkLocationID == "" {
		return nil, nil, nil
	}
	locationID := loc.WorkLocationID

	if cached, ok := perPage[locationID]; ok {
		return cached, nil, nil
	}

	wl, found, err := session.GetJSON[client.WorkLocation](ctx, ss, locationID, sessions.WithPrefix(workLocationsSessionPrefix))
	if err != nil {
		return nil, nil, fmt.Errorf("baton-rippling: failed to read work location %s from session store: %w", locationID, err)
	}
	if found {
		perPage[locationID] = &wl
		return &wl, nil, nil
	}

	fetched, rl, err := o.fetchWorkLocation(ctx, locationID)
	if err != nil {
		return nil, rl, err
	}
	if err := session.SetJSON(ctx, ss, locationID, *fetched, sessions.WithPrefix(workLocationsSessionPrefix)); err != nil {
		return nil, rl, fmt.Errorf("baton-rippling: failed to store work location %s in session store: %w", locationID, err)
	}
	perPage[locationID] = fetched
	return fetched, rl, nil
}

// resolveWorkEmail returns the user's work email for use as a login alias:
// worker.WorkEmail takes precedence, then a WORK-typed entry in user.Emails.
func resolveWorkEmail(user client.User, worker *client.Worker) string {
	if worker != nil && worker.WorkEmail != "" {
		return worker.WorkEmail
	}
	if e := getWorkEmail(user.Emails); e != nil && e.Value != "" && strings.EqualFold(e.Type, workType) {
		return e.Value
	}
	return ""
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
	var annos annotations.Annotations
	worker, found, err := session.GetJSON[client.Worker](ctx, opts.Session, r.Id.Resource, sessions.WithPrefix(workersSessionPrefix))
	if err != nil {
		return nil, nil, fmt.Errorf("baton-rippling: failed to get worker for user %s from session store: %w", r.Id.Resource, err)
	}
	if !found {
		recovered, rl, err := o.fetchWorker(ctx, r.Id.Resource)
		annos = *annos.WithRateLimiting(rl)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, err
		}
		if recovered == nil {
			ctxzap.Extract(ctx).Debug("no worker exists for user, skipping team grants", zap.String("user_id", r.Id.Resource))
			return nil, &resource.SyncOpResults{Annotations: annos}, nil
		}
		worker = *recovered
	}
	if worker.Status == "TERMINATED" {
		ctxzap.Extract(ctx).Debug("worker is terminated, skipping team grants", zap.String("user_id", r.Id.Resource))
		return nil, &resource.SyncOpResults{Annotations: annos}, nil
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

	return rv, &resource.SyncOpResults{Annotations: annos}, nil
}

// Fallback for users the /workers pagination missed.
//
// Someone who has been rehired may have more than one worker record. We
// use the most recently updated one.
func (o *userBuilder) fetchWorker(ctx context.Context, userID string) (*client.Worker, *v2.RateLimitDescription, error) {
	resp, rl, err := o.client.GetWorkersByUserID(ctx, userID)
	if err != nil {
		return nil, rl, fmt.Errorf("baton-rippling: on-demand worker fetch failed for user %s: %w", userID, err)
	}
	if len(resp.Results) == 0 {
		return nil, rl, nil
	}
	chosen := resp.Results[0]
	for _, w := range resp.Results[1:] {
		if w.UpdatedAt > chosen.UpdatedAt {
			chosen = w
		}
	}
	return &chosen, rl, nil
}

func (o *userBuilder) fetchWorkLocation(ctx context.Context, locationID string) (*client.WorkLocation, *v2.RateLimitDescription, error) {
	wl, rl, err := o.client.GetWorkLocationByID(ctx, locationID)
	if err != nil {
		return nil, rl, fmt.Errorf("baton-rippling: on-demand work location fetch failed for %s: %w", locationID, err)
	}
	return wl, rl, nil
}

// newUserBuilder returns the user syncer. Users have no entitlements of their
// own, and their only grants are cross-type team grants, so when team is
// excluded from the sync the grants pass is skipped too.
func newUserBuilder(client *client.Client, expandWorkLocations bool, customFieldNames []string, skipTeamResourceType bool) *userBuilder {
	rt := proto.Clone(userResourceType).(*v2.ResourceType)
	annos := annotations.Annotations(rt.GetAnnotations())
	if skipTeamResourceType {
		annos.Update(&v2.SkipEntitlementsAndGrants{})
	} else {
		annos.Update(&v2.SkipEntitlements{})
	}
	rt.Annotations = annos

	return &userBuilder{
		resourceType:        rt,
		client:              client,
		expandWorkLocations: expandWorkLocations,
		customFieldNames:    customFieldNames,
	}
}
