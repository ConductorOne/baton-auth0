package main

import (
	"context"

	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/spf13/viper"
)

// ConfigurationFields defines the external configuration required for the connector to run.
var ConfigurationFields = []field.SchemaField{}

// ValidateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func ValidateConfig(ctx context.Context, v *viper.Viper) error {
	return nil
}
