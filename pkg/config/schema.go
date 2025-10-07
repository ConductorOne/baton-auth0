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
	SyncUsersByJob = field.BoolField(
		"sync-users-by-job",
		field.WithDescription("Sync users by job (only applicable for Auth0 tenants with more than 1000 users https://auth0.com/docs/users/search/v3/view-search-results-by-page#limitation)"),
	)
	SyncUsersByJobLimit = field.IntField(
		"sync-users-by-job-limit",
		field.WithDescription("Number of users to fetch per job (only applicable if sync-users-by-job is true)"),
	)
	// ConfigurationFields defines the external configuration required for the connector to run.
	ConfigurationFields = []field.SchemaField{
		BaseUrlField,
		ClientIdField,
		ClientSecretField,
		SyncPermissions,
		SyncUsersByJob,
		SyncUsersByJobLimit,
	}

	ConfigurationSchema = field.NewConfiguration(ConfigurationFields)
)
