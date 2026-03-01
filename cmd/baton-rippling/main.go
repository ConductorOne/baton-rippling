//go:build !generate

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-rippling/pkg/client"
	cfg "github.com/conductorone/baton-rippling/pkg/config"
	"github.com/conductorone/baton-rippling/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

var version = "dev"

func main() {
	ctx := context.Background()

	_, cmd, err := config.DefineConfigurationV2(
		ctx,
		"baton-rippling",
		getConnector,
		cfg.Config,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Connector{}),
		connectorrunner.WithSessionStoreEnabled(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, config *cfg.Rippling, runTimeOpts cli.RunTimeOpts) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)
	if err := field.Validate(cfg.Config, config); err != nil {
		return nil, err
	}

	cb, err := connector.New(ctx, config.ApiToken, config.BaseUrl, client.ExpandOptions{
		Department:     config.ExpandDepartment,
		EmploymentType: config.ExpandEmploymentType,
		Level:          config.ExpandLevel,
	}, config.ExpandWorkLocations)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}
	c, err := connectorbuilder.NewConnector(ctx, cb, connectorbuilder.WithSessionStore(runTimeOpts.SessionStore))
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}
	return c, nil
}
