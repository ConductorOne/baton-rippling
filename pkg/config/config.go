package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	// Add the SchemaFields for the Config.
	ApiToken = field.StringField("api-token",
		field.WithDisplayName("API token"),
		field.WithDescription("The API token for the Rippling connector. This is used to authenticate API requests."),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)
	ConfigurationFields = []field.SchemaField{ApiToken}

	// FieldRelationships defines relationships between the ConfigurationFields that can be automatically validated.
	// For example, a username and password can be required together, or an access token can be
	// marked as mutually exclusive from the username password pair.
	FieldRelationships = []field.SchemaFieldRelationship{}
)

//go:generate go run -tags=generate ./gen
var Config = field.NewConfiguration(
	ConfigurationFields,
	field.WithConstraints(FieldRelationships...),
	field.WithConnectorDisplayName("Rippling"),
	field.WithHelpUrl("/docs/baton/rippling"),
	field.WithIconUrl("/static/app-icons/rippling.svg"),
)
