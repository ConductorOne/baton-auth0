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
