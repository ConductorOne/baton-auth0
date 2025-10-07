package client

import "time"

type AuthRequest struct {
	Audience     string `json:"audience"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
	TokenType   string `json:"token_type"`
}

type Organization struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type OrganizationMembersResponse struct {
	Members []User `json:"members"`
	PaginatedResponse
}

type OrganizationsResponse struct {
	Organizations []Organization `json:"organizations"`
	PaginatedResponse
}

type PaginatedResponse struct {
	Start int `json:"start"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RolesResponse struct {
	PaginatedResponse
	Roles []Role `json:"roles"`
}

type RolesUsersResponse struct {
	PaginatedResponse
	Users []User `json:"users"`
}

type User struct {
	CreatedAt     time.Time        `json:"created_at"`
	Email         string           `json:"email"`
	EmailVerified bool             `json:"email_verified"`
	Identities    []UserIdentities `json:"identities"`
	Name          string           `json:"name"`
	Nickname      string           `json:"nickname"`
	Picture       string           `json:"picture"`
	UpdatedAt     time.Time        `json:"updated_at"`
	UserId        string           `json:"user_id"`
}

type UserIdentities struct {
	Connection string `json:"connection"`
	IsSocial   bool   `json:"isSocial"`
	Provider   string `json:"provider"`
	UserId     string `json:"user_id"`
}

type UsersResponse struct {
	PaginatedResponse
	Length int    `json:"length"`
	Users  []User `json:"users"`
}

type ResourceServerScope struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type ResourceServer struct {
	Id                                        string                `json:"id"`
	Name                                      string                `json:"name"`
	IsSystem                                  bool                  `json:"is_system"`
	Identifier                                string                `json:"identifier"`
	Scopes                                    []ResourceServerScope `json:"scopes"`
	SigningAlg                                string                `json:"signing_alg"`
	AllowOfflineAccess                        bool                  `json:"allow_offline_access"`
	SkipConsentForVerifiableFirstPartyClients bool                  `json:"skip_consent_for_verifiable_first_party_clients"`
	TokenLifetime                             int                   `json:"token_lifetime"`
	TokenLifetimeForWeb                       int                   `json:"token_lifetime_for_web"`
	EnforcePolicies                           bool                  `json:"enforce_policies"`
	TokenDialect                              string                `json:"token_dialect"`
	ConsentPolicy                             string                `json:"consent_policy"`
}

type ResourceServerResponse struct {
	PaginatedResponse
	ResourceServers []*ResourceServer `json:"resource_servers"`
}

type RolePermission struct {
	PermissionName           string `json:"permission_name"`
	Description              string `json:"description"`
	ResourceServerName       string `json:"resource_server_name"`
	ResourceServerIdentifier string `json:"resource_server_identifier"`
}

type JobField struct {
	Name string `json:"name"`
}

type Job struct {
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Format    string     `json:"format"`
	Limit     int        `json:"limit"`
	Fields    []JobField `json:"fields"`
	CreatedAt time.Time  `json:"created_at"`
	Id        string     `json:"id"`
	Location  string     `json:"location,omitempty"`
}
