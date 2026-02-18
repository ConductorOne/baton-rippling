package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const teamMembership = "member"

type teamBuilder struct {
	client *client.Client
}

func (o *teamBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return teamResourceType
}

func teamResource(team client.Team) (*v2.Resource, error) {
	return resourceSdk.NewGroupResource(
		team.Name,
		teamResourceType,
		team.ID,
		[]resourceSdk.GroupTraitOption{},
	)
}

// List returns all the teams from the database as resource objects.
// Teams include a TeamTrait because they are the 'shape' of a standard team.
func (o *teamBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, opts resource.SyncOpAttrs) ([]*v2.Resource, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	teamsResponse, ratelimitData, err := o.client.ListTeams(ctx, opts.PageToken.Token)
	annos = *annos.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list teams: %w", err)
	}

	rv := make([]*v2.Resource, 0, len(teamsResponse.Results))
	for _, team := range teamsResponse.Results {
		r, err := teamResource(team)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to convert team %s to resource: %w", team.ID, err)
		}
		rv = append(rv, r)
	}

	return rv, &resource.SyncOpResults{
		NextPageToken: teamsResponse.NextLink,
		Annotations:   annos,
	}, nil
}

func (o *teamBuilder) Entitlements(_ context.Context, r *v2.Resource, _ resource.SyncOpAttrs) ([]*v2.Entitlement, *resource.SyncOpResults, error) {
	return []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			r,
			teamMembership,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDescription(fmt.Sprintf("Member of %s team", r.DisplayName)),
			entitlement.WithDisplayName(fmt.Sprintf("Member of %s team", r.DisplayName)),
		),
	}, nil, nil
}

func (o *teamBuilder) Grants(ctx context.Context, r *v2.Resource, opts resource.SyncOpAttrs) ([]*v2.Grant, *resource.SyncOpResults, error) {
	var annos annotations.Annotations
	res, ratelimitDescription, err := o.client.ListWorkers(ctx, opts.PageToken.Token)
	annos = *annos.WithRateLimiting(ratelimitDescription)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to list workers: %w", err)
	}

	rv := []*v2.Grant{}
	for _, worker := range res.Results {
		if worker.Status == "TERMINATED" {
			continue
		}
		for _, teamId := range worker.TeamsID {
			if teamId != r.Id.Resource {
				continue
			}

			principalId, err := resourceSdk.NewResourceID(userResourceType, worker.UserID)
			if err != nil {
				return nil, &resource.SyncOpResults{Annotations: annos}, fmt.Errorf("baton-rippling: failed to create resource ID for user %s: %w", worker.UserID, err)
			}
			rv = append(rv, grant.NewGrant(
				r,
				teamMembership,
				principalId,
			))
		}
	}
	return rv, &resource.SyncOpResults{
		NextPageToken: res.NextLink,
		Annotations:   annos,
	}, nil
}

func newTeamBuilder(client *client.Client) *teamBuilder {
	return &teamBuilder{
		client: client,
	}
}
