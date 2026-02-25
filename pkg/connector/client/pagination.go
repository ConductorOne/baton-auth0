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
	Since           string `json:"since,omitempty"`
	Until           string `json:"until,omitempty"`
}

// ParsePaginationToken - takes as pagination token and returns page, limit,
// and `pagingRequestId` in that order.
func ParsePaginationToken(pToken *pagination.Token) (
	int,
	int,
	string,
	string,
	string,
	error,
) {
	var (
		limit           = PageSizeDefault
		page            = 0
		pagingRequestId = ""
		since           = "*"
		until           = time.Now().UTC().Format(time.RFC3339Nano)
	)

	if pToken == nil {
		return page, limit, pagingRequestId, since, until, nil
	}

	if pToken.Size > 0 {
		limit = pToken.Size
	}

	if pToken.Token == "" {
		return page, limit, pagingRequestId, since, until, nil
	}

	var parsed Pagination
	err := json.Unmarshal([]byte(pToken.Token), &parsed)
	if err != nil {
		return 0, 0, "", "", "", err
	}

	page = parsed.Page
	pagingRequestId = parsed.PagingRequestId
	since = parsed.Since
	until = parsed.Until
	return page, limit, pagingRequestId, since, until, nil
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

// GetNextToken given a limit and page that were used to fetch _this_ page of
// data, and total number of resources, return the next pagination token as a
// string.
func GetNextToken(
	page int,
	limit int,
	total int,
	newestCreatedAt *time.Time,
) string {
	nextPage := page + 1
	nextOffset := nextPage * limit

	if nextOffset >= total {
		return ""
	}

	bytes, err := json.Marshal(
		Pagination{
			Page:  nextPage,
			Since: newestCreatedAt.UTC().Format(time.RFC3339Nano),
		},
	)
	if err != nil {
		return ""
	}

	return string(bytes)
}
