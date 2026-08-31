package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// doGet performs a GET request to the given URL (or nextLink if non-empty),
// deserializes the JSON response into target, and returns rate limit data.
func (c *Client) doGet(ctx context.Context, baseURL, nextLink string, target any) (*v2.RateLimitDescription, error) {
	url := baseURL
	if nextLink != "" {
		url = nextLink
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var ratelimitData v2.RateLimitDescription
	res, err := c.Do(
		req,
		uhttp.WithJSONResponse(target),
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		if res != nil {
			defer res.Body.Close()
			logBody(ctx, res.Body)
		}
		return &ratelimitData, fmt.Errorf("request to %s failed: %w", baseURL, err)
	}
	defer res.Body.Close()

	return &ratelimitData, nil
}

func (c *Client) ListWorkers(ctx context.Context, nextLink string) (*WorkersResponse, *v2.RateLimitDescription, error) {
	// Build the base URL with expand params so that doGet handles the rest.
	// The expand param is only needed on the initial request; next_link URLs
	// already include it.
	u, _ := url.Parse(c.workersURL())
	if expand := c.buildExpandParam(); expand != "" {
		q := u.Query()
		q.Set("expand", expand)
		u.RawQuery = q.Encode()
	}
	baseURL := u.String()

	var workers WorkersResponse
	rl, err := c.doGet(ctx, baseURL, nextLink, &workers)
	if err != nil {
		return nil, rl, fmt.Errorf("failed to list workers: %w", err)
	}
	return &workers, rl, nil
}

func (c *Client) GetWorkersByUserID(ctx context.Context, userID string) (*WorkersResponse, *v2.RateLimitDescription, error) {
	u, _ := url.Parse(c.workersURL())
	q := u.Query()
	q.Set("filter", fmt.Sprintf("user_id eq '%s'", userID))
	if expand := c.buildExpandParam(); expand != "" {
		q.Set("expand", expand)
	}
	u.RawQuery = q.Encode()

	var workers WorkersResponse
	rl, err := c.doGet(ctx, u.String(), "", &workers)
	if err != nil {
		return nil, rl, fmt.Errorf("failed to fetch worker by user_id %s: %w", userID, err)
	}
	return &workers, rl, nil
}

func (c *Client) ListTeams(ctx context.Context, nextLink string) (*TeamsResponse, *v2.RateLimitDescription, error) {
	var teams TeamsResponse
	rl, err := c.doGet(ctx, c.teamsURL(), nextLink, &teams)
	if err != nil {
		return nil, rl, fmt.Errorf("failed to list teams: %w", err)
	}
	return &teams, rl, nil
}

func (c *Client) ListWorkLocations(ctx context.Context, nextLink string) (*WorkLocationsResponse, *v2.RateLimitDescription, error) {
	var workLocations WorkLocationsResponse
	rl, err := c.doGet(ctx, c.workLocationsURL(), nextLink, &workLocations)
	if err != nil {
		return nil, rl, fmt.Errorf("failed to list work locations: %w", err)
	}
	return &workLocations, rl, nil
}

func (c *Client) GetWorkLocationByID(ctx context.Context, locationID string) (*WorkLocation, *v2.RateLimitDescription, error) {
	u, err := url.JoinPath(c.workLocationsURL(), locationID+"/")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build work location URL for %s: %w", locationID, err)
	}
	var workLocation WorkLocation
	rl, err := c.doGet(ctx, u, "", &workLocation)
	if err != nil {
		return nil, rl, fmt.Errorf("failed to fetch work location %s: %w", locationID, err)
	}
	return &workLocation, rl, nil
}

