package client

import (
	"context"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
)

type Client struct {
	*uhttp.BaseHttpClient
}

func New(ctx context.Context, apiToken string) (*Client, error) {
	client, err := uhttp.NewBearerAuth(apiToken).GetClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, uhttp.WrapErrors(codes.Unauthenticated, "rippling-connector: failed to create HTTP client", err)
	}

	return &Client{
		BaseHttpClient: uhttp.NewBaseHttpClient(client),
	}, nil
}
