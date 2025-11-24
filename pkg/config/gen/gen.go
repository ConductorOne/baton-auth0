package main

import (
	cfg "github.com/conductorone/baton-auth0/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("auth0", cfg.Config)
}