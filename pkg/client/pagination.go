package client

import (
	"encoding/json"
	"time"

	"github.com/conductorone/baton-sdk/pkg/pagination"
)

const PageSizeDefault = 100

// Auth0UserSearchMaxResults is the hard limit imposed by Auth0's User Search API.
// Paginating beyond this limit results in a 400 error.
// See https://auth0.com/docs/manage-users/user-search/view-search-results-by-page#limitation.
const Auth0UserSearchMaxResults = 1000

type Pagination struct {
	PagingRequestId string `json:"pagingRequestId"`
	Page            int    `json:"page"`
}

type UserPagination struct {
	Page                int        `json:"page"`
	Since               string     `json:"since,omitempty"`
	Until               string     `json:"until,omitempty"`
	NewestUserCreatedAt *time.Time `json:"newestUserCreatedAt,omitempty"`
}

// ParseUserPaginationToken - takes as pagination token and returns page, limit,
// also it includes since and until dates to search users, and the date the newest user was created.
func ParseUserPaginationToken(pToken *pagination.Token) (
	int,
	int,
	string,
	string,
	*time.Time,
	error,
) {
	var (
		limit = PageSizeDefault
		page  = 0
		since = "*"

		// Until never gets updated. Always should use the last possible date.
		until = time.Now().UTC().Format(time.RFC3339Nano)
	)
	var newestUserCreationDate *time.Time

	if pToken == nil {
		return page, limit, since, until, newestUserCreationDate, nil
	}

	if pToken.Size > 0 {
		limit = pToken.Size
	}

	if pToken.Token == "" {
		return page, limit, since, until, newestUserCreationDate, nil
	}

	var parsed UserPagination
	err := json.Unmarshal([]byte(pToken.Token), &parsed)
	if err != nil {
		return 0, 0, "", "", nil, err
	}

	page = parsed.Page
	newestUserCreationDate = parsed.NewestUserCreatedAt
	if parsed.Since != "" {
		since = parsed.Since
	}

	return page, limit, since, until, newestUserCreationDate, nil
}

// ParsePaginationToken - takes as pagination token and returns page, limit,
// and `pagingRequestId` in that order.
func ParsePaginationToken(pToken *pagination.Token) (
	int,
	int,
	string,
	error,
) {
	var (
		limit           = PageSizeDefault
		page            = 0
		pagingRequestId = ""
	)

	if pToken == nil {
		return page, limit, pagingRequestId, nil
	}

	if pToken.Size > 0 {
		limit = pToken.Size
	}

	if pToken.Token == "" {
		return page, limit, pagingRequestId, nil
	}

	var parsed Pagination
	err := json.Unmarshal([]byte(pToken.Token), &parsed)
	if err != nil {
		return 0, 0, "", err
	}

	page = parsed.Page
	pagingRequestId = parsed.PagingRequestId
	return page, limit, pagingRequestId, nil
}

func ParsePaginationTokenString(pToken string) (
	int,
	int,
	string,
	error,
) {
	var (
		limit           = PageSizeDefault
		page            = 0
		pagingRequestId = ""
	)

	if pToken == "" {
		return page, limit, pagingRequestId, nil
	}

	var parsed Pagination
	err := json.Unmarshal([]byte(pToken), &parsed)
	if err != nil {
		return 0, 0, "", err
	}

	page = parsed.Page
	pagingRequestId = parsed.PagingRequestId
	return page, limit, pagingRequestId, nil
}

// RoleUserCheckpointPagination holds the opaque checkpoint token used by Auth0's
// checkpoint-based pagination for the GET /api/v2/roles/{id}/users endpoint.
type RoleUserCheckpointPagination struct {
	From string `json:"from,omitempty"`
}

// ParseRoleUserCheckpointToken extracts the checkpoint "from" value from a
// serialized RoleUserCheckpointPagination token. Returns "" for the first page.
func ParseRoleUserCheckpointToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	var parsed RoleUserCheckpointPagination
	if err := json.Unmarshal([]byte(token), &parsed); err != nil {
		return "", err
	}
	return parsed.From, nil
}

// GetNextRoleUserCheckpointToken serializes the checkpoint token returned by
// Auth0 into a pagination token string. Returns "" when next is empty (no more pages).
func GetNextRoleUserCheckpointToken(next string) string {
	if next == "" {
		return ""
	}
	bytes, err := json.Marshal(RoleUserCheckpointPagination{From: next})
	if err != nil {
		return ""
	}
	return string(bytes)
}

// GetNextToken given a limit and page that were used to fetch _this_ page of
// data, and total number of resources, return the next pagination token as a
// string.
func GetNextToken(
	page int,
	limit int,
	total int,
) string {
	nextPage := page + 1
	nextOffset := nextPage * limit

	if nextOffset >= total {
		return ""
	}

	bytes, err := json.Marshal(Pagination{
		Page: nextPage,
	})
	if err != nil {
		return ""
	}

	return string(bytes)
}

func GetNextUsersToken(
	page int,
	limit int,
	total int,
	since string,
	newestCreatedAt *time.Time,
) (string, error) {
	nextPage := page + 1
	nextOffset := nextPage * limit

	if nextOffset >= Auth0UserSearchMaxResults {
		if newestCreatedAt == nil {
			return "", nil
		}

		nextSince := newestCreatedAt.UTC().Format(time.RFC3339Nano)
		if nextSince == since {
			// No forward progress — stop to avoid infinite loop
			return "", nil
		}
		
		nextPage = 0
		since = nextSince
	} else if nextOffset >= total {
		return "", nil
	}

	bytes, err := json.Marshal(UserPagination{
		Page:                nextPage,
		Since:               since,
		NewestUserCreatedAt: newestCreatedAt,
	})
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
