package client

import (
	"encoding/json"

	"github.com/conductorone/baton-sdk/pkg/pagination"
)

const PageSizeDefault = 100

type Pagination struct {
	PagingRequestId string `json:"pagingRequestId"`
	Page            int    `json:"page"`
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

	bytes, err := json.Marshal(
		Pagination{
			Page: nextPage,
		},
	)
	if err != nil {
		return ""
	}

	return string(bytes)
}
