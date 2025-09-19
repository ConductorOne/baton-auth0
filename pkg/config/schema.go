package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	BaseUrlField = field.StringField(
		"auth0-base-url",
		field.WithDescription("Base URL of the API"),
		field.WithRequired(true),
	)
	ClientIdField = field.StringField(
		"auth0-client-id",
		field.WithDescription("App client ID"),
		field.WithRequired(true),
	)
	ClientSecretField = field.StringField(
		"auth0-client-secret",
		field.WithDescription("App client secret"),
		field.WithRequired(true),
	)
	SyncPermissions = field.BoolField(
		"sync-permissions",
		field.WithDescription("Sync permissions"),
	)
	// ConfigurationFields defines the external configuration required for the connector to run.
	ConfigurationFields = []field.SchemaField{
		BaseUrlField,
		ClientIdField,
		ClientSecretField,
		SyncPermissions,
	}

	ConfigurationSchema = field.NewConfiguration(ConfigurationFields)
)
