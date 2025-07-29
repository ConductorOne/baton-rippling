package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

func (c *Client) ListWorkers(ctx context.Context, nextLink string) (*WorkersResponse, *v2.RateLimitDescription, error) {
	url := WorkersURL
	if nextLink != "" {
		url = nextLink
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
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
			logBody(ctx, res.Body)
		}
		return nil, &ratelimitData, fmt.Errorf("failed to list workers: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return &workers, &ratelimitData, nil
}

func (c *Client) ListTeams(ctx context.Context, nextLink string) (*TeamsResponse, *v2.RateLimitDescription, error) {
	url := TeamsURL
	if nextLink != "" {
		url = nextLink
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var ratelimitData v2.RateLimitDescription
	var teams TeamsResponse
	res, err := c.Do(
		req,
		uhttp.WithJSONResponse(&teams),
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		if res != nil {
			logBody(ctx, res.Body)
		}
		return nil, &ratelimitData, fmt.Errorf("failed to list teams: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return &teams, &ratelimitData, nil
}

func (c *Client) ListUsers(ctx context.Context, nextLink string) (*UsersResponse, *v2.RateLimitDescription, error) {
	url := UsersURL
	if nextLink != "" {
		url = nextLink
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var ratelimitData v2.RateLimitDescription
	var usersResponse UsersResponse
	res, err := c.Do(
		req,
		uhttp.WithJSONResponse(&usersResponse),
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		if res != nil {
			logBody(ctx, res.Body)
		}
		return nil, &ratelimitData, fmt.Errorf("failed to list users: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return &usersResponse, &ratelimitData, nil
}
