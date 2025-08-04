package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-rippling/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
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
func (o *teamBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	teamsResponse, ratelimitData, err := o.client.ListTeams(ctx, pToken.Token)
	annotations = *annotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-rippling: failed to list teams: %w", err)
	}

	rv := []*v2.Resource{}
	for _, team := range teamsResponse.Results {
		resource, err := teamResource(team)
		if err != nil {
			return nil, "", annotations, fmt.Errorf("baton-rippling: failed to convert team %s to resource: %w", team.ID, err)
		}
		rv = append(rv, resource)
	}

	return rv, teamsResponse.NextLink, annotations, nil
}

func (o *teamBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			resource,
			teamMembership,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDescription(fmt.Sprintf("Member of %s team", resource.DisplayName)),
			entitlement.WithDisplayName(fmt.Sprintf("Member of %s team", resource.DisplayName)),
		),
	}, "", nil, nil
}

func (o *teamBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	res, ratelimitDescription, err := o.client.ListWorkers(ctx, pToken.Token)
	annotations = *annotations.WithRateLimiting(ratelimitDescription)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-rippling: failed to list workers: %w", err)
	}

	rv := []*v2.Grant{}
	for _, worker := range res.Results {
		if worker.Status == "TERMINATED" {
			continue
		}
		for _, teamId := range worker.TeamsID {
			if teamId != resource.Id.Resource {
				continue
			}

			principalId, err := resourceSdk.NewResourceID(userResourceType, worker.UserID)
			if err != nil {
				return nil, "", annotations, fmt.Errorf("baton-rippling: failed to create resource ID for user %s: %w", worker.UserID, err)
			}
			rv = append(rv, grant.NewGrant(
				resource,
				teamMembership,
				principalId,
			))
		}
	}
	return rv, res.NextLink, nil, nil
}

func newTeamBuilder(client *client.Client) *teamBuilder {
	return &teamBuilder{
		client: client,
	}
}
