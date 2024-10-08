package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	BaseUrlField = field.StringField(
		"base-url",
		field.WithDescription("Base URL of the API"),
		field.WithRequired(true),
	)
	ClientIdField = field.StringField(
		"oauth0-client-id",
		field.WithDescription("App client ID"),
		field.WithRequired(true),
	)
	ClientSecretField = field.StringField(
		"oauth0-client-secret",
		field.WithDescription("App client secret"),
		field.WithRequired(true),
	)
	// ConfigurationFields defines the external configuration required for the connector to run.
	ConfigurationFields = []field.SchemaField{
		BaseUrlField,
		ClientIdField,
		ClientSecretField,
	}

	ConfigurationSchema = field.NewConfiguration(ConfigurationFields)
)
