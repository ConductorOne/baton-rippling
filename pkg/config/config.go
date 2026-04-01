package config

import (
	"github.com/conductorone/baton-rippling/pkg/client"
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
	BaseURL = field.StringField("base-url",
		field.WithDisplayName("Base URL"),
		field.WithDescription("The base URL for the Rippling API."),
		field.WithDefaultValue(client.DefaultBaseURL),
		field.WithHidden(true),
		field.WithExportTarget(field.ExportTargetCLIOnly),
	)
	ExpandDepartment = field.BoolField("expand-department",
		field.WithDisplayName("Expand department"),
		field.WithDescription("Include department data in worker sync. Requires the departments.read scope."),
	)
	ExpandEmploymentType = field.BoolField("expand-employment-type",
		field.WithDisplayName("Expand employment type"),
		field.WithDescription("Include employment type data in worker sync. Requires the employment-types.read scope."),
	)
	ExpandLevel = field.BoolField("expand-level",
		field.WithDisplayName("Expand level"),
		field.WithDescription("Include level data in worker sync. Requires the levels.read scope."),
	)
	ExpandWorkLocations = field.BoolField("expand-work-locations",
		field.WithDisplayName("Expand work locations"),
		field.WithDescription("Include work location name and address in user profiles. Requires the work-locations.read scope."),
	)
	CustomFields = field.StringSliceField("custom-fields",
		field.WithDisplayName("Custom fields"),
		field.WithDescription("List of custom field names to sync from the Rippling Workers endpoint. Only fields with names matching this list (case-insensitive) will be included in user profiles."),
	)
	ConfigurationFields = []field.SchemaField{
		ApiToken,
		BaseURL,
		ExpandDepartment,
		ExpandEmploymentType,
		ExpandLevel,
		ExpandWorkLocations,
		CustomFields,
	}

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
