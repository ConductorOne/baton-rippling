package client

import (
	"context"
	"fmt"
	"net/http"

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
		return nil, fmt.Errorf("baton-rippling: failed to create request: %w", err)
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
		return &ratelimitData, fmt.Errorf("baton-rippling: request to %s failed: %w", baseURL, err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return &ratelimitData, fmt.Errorf("baton-rippling: unexpected status code: %d", res.StatusCode)
	}

	return &ratelimitData, nil
}

func (c *Client) ListWorkers(ctx context.Context, nextLink string) (*WorkersResponse, *v2.RateLimitDescription, error) {
	url := c.workersURL()
	if nextLink != "" {
		url = nextLink
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-rippling: failed to create request: %w", err)
	}

	// Only add expand params on the initial request; next_link URLs already include them.
	if nextLink == "" {
		if expand := c.buildExpandParam(); expand != "" {
			q := req.URL.Query()
			q.Set("expand", expand)
			req.URL.RawQuery = q.Encode()
		}
	}

	var ratelimitData v2.RateLimitDescription
	var workers WorkersResponse
	res, err := c.Do(
		req,
		uhttp.WithJSONResponse(&workers),
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		if res != nil {
			defer res.Body.Close()
			logBody(ctx, res.Body)
		}
		return nil, &ratelimitData, fmt.Errorf("baton-rippling: failed to list workers: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, fmt.Errorf("baton-rippling: unexpected status code: %d", res.StatusCode)
	}

	return &workers, &ratelimitData, nil
}

func (c *Client) ListTeams(ctx context.Context, nextLink string) (*TeamsResponse, *v2.RateLimitDescription, error) {
	var teams TeamsResponse
	rl, err := c.doGet(ctx, c.teamsURL(), nextLink, &teams)
	if err != nil {
		return nil, rl, fmt.Errorf("baton-rippling: failed to list teams: %w", err)
	}
	return &teams, rl, nil
}

func (c *Client) ListWorkLocations(ctx context.Context, nextLink string) (*WorkLocationsResponse, *v2.RateLimitDescription, error) {
	var workLocations WorkLocationsResponse
	rl, err := c.doGet(ctx, c.workLocationsURL(), nextLink, &workLocations)
	if err != nil {
		return nil, rl, fmt.Errorf("baton-rippling: failed to list work locations: %w", err)
	}
	return &workLocations, rl, nil
}

func (c *Client) ListUsers(ctx context.Context, nextLink string) (*UsersResponse, *v2.RateLimitDescription, error) {
	var usersResponse UsersResponse
	rl, err := c.doGet(ctx, c.usersURL(), nextLink, &usersResponse)
	if err != nil {
		return nil, rl, fmt.Errorf("baton-rippling: failed to list users: %w", err)
	}
	return &usersResponse, rl, nil
}
