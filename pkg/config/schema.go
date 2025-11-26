//go:generate go run ./gen
package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	BaseUrlField = field.StringField(
		"auth0-base-url",
		field.WithDisplayName("Base URL"),
		field.WithDescription("Base URL of the Auth0 API (e.g., https://your-tenant.auth0.com)"),
		field.WithPlaceholder("https://your-tenant.auth0.com"),
		field.WithRequired(true),
	)
	ClientIdField = field.StringField(
		"auth0-client-id",
		field.WithDisplayName("Client ID"),
		field.WithDescription("Auth0 Machine-to-Machine application client ID"),
		field.WithPlaceholder("your_client_id"),
		field.WithRequired(true),
	)
	ClientSecretField = field.StringField(
		"auth0-client-secret",
		field.WithDisplayName("Client Secret"),
		field.WithDescription("Auth0 Machine-to-Machine application client secret"),
		field.WithIsSecret(true),
		field.WithRequired(true),
	)
	SyncPermissions = field.BoolField(
		"sync-permissions",
		field.WithDisplayName("Sync Permissions"),
		field.WithDescription("Sync permissions along with roles and users"),
	)
)

// ConfigurationFields defines the external configuration required for the connector to run.
var ConfigurationFields = []field.SchemaField{
	BaseUrlField,
	ClientIdField,
	ClientSecretField,
	SyncPermissions,
}

// Config defines the configuration for the Auth0 connector.
var Config = field.NewConfiguration(
	ConfigurationFields,
	field.WithConnectorDisplayName("Auth0"),
	field.WithHelpUrl("/docs/baton/auth0"),
	field.WithIconUrl("/static/app-icons/auth0.svg"),
)
