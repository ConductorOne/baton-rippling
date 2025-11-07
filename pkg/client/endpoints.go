package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
)

func (c *Client) ListWorkers(ctx context.Context, nextLink string) (*WorkersResponse, *v2.RateLimitDescription, error) {
	url := WorkersURL
	if nextLink != "" {
		url = nextLink
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("rippling-connector: failed to create request for workers: %w", err)
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
		return nil, &ratelimitData, uhttp.WrapErrors(codes.Unavailable, "rippling-connector: failed to list workers from API", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, uhttp.WrapErrors(httpToGRPCCode(res.StatusCode), fmt.Sprintf("rippling-connector: unexpected status code listing workers: %d", res.StatusCode))
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
		return nil, nil, fmt.Errorf("rippling-connector: failed to create request for teams: %w", err)
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
		return nil, &ratelimitData, uhttp.WrapErrors(codes.Unavailable, "rippling-connector: failed to list teams from API", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, uhttp.WrapErrors(httpToGRPCCode(res.StatusCode), fmt.Sprintf("rippling-connector: unexpected status code listing teams: %d", res.StatusCode))
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
		return nil, nil, fmt.Errorf("rippling-connector: failed to create request for users: %w", err)
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
		return nil, &ratelimitData, uhttp.WrapErrors(codes.Unavailable, "rippling-connector: failed to list users from API", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logBody(ctx, res.Body)
		return nil, &ratelimitData, uhttp.WrapErrors(httpToGRPCCode(res.StatusCode), fmt.Sprintf("rippling-connector: unexpected status code listing users: %d", res.StatusCode))
	}

	return &usersResponse, &ratelimitData, nil
}
