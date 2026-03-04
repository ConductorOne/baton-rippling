package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

type ExpandOptions struct {
	Department     bool
	EmploymentType bool
	Level          bool
}

type Client struct {
	*uhttp.BaseHttpClient
	baseURL       string
	expandOptions ExpandOptions
}

func New(ctx context.Context, apiToken string, baseURL string, expandOpts ExpandOptions) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	client, err := uhttp.NewBearerAuth(apiToken).GetClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, fmt.Errorf("baton-rippling: failed to create HTTP client: %w", err)
	}

	return &Client{
		BaseHttpClient: uhttp.NewBaseHttpClient(client),
		baseURL:        baseURL,
		expandOptions:  expandOpts,
	}, nil
}

func (c *Client) usersURL() string         { return c.baseURL + usersPath }
func (c *Client) teamsURL() string         { return c.baseURL + teamsPath }
func (c *Client) workersURL() string       { return c.baseURL + workersPath }
func (c *Client) workLocationsURL() string { return c.baseURL + workLocationsPath }

func (c *Client) buildExpandParam() string {
	// Manager is always expanded since it only requires the workers.read scope
	// which is already required for the base workers endpoint.
	fields := []string{"manager"}
	if c.expandOptions.Department {
		fields = append(fields, "department")
	}
	if c.expandOptions.EmploymentType {
		fields = append(fields, "employment_type")
	}
	if c.expandOptions.Level {
		fields = append(fields, "level")
	}
	return strings.Join(fields, ",")
}
