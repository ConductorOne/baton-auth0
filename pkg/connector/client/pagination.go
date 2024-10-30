package client

import (
	"encoding/json"

	"github.com/conductorone/baton-sdk/pkg/pagination"
)

const PageSizeDefault = 100

type Pagination struct {
	PagingRequestId string `json:"pagingRequestId"`
	Offset          int    `json:"offset"`
}

// ParsePaginationToken - takes as pagination token and returns offset, limit,
// and `pagingRequestId` in that order.
func ParsePaginationToken(pToken *pagination.Token) (
	int,
	int,
	string,
	error,
) {
	var (
		limit           = PageSizeDefault
		offset          = 0
		pagingRequestId = ""
	)

	if pToken != nil {
		if pToken.Size > 0 {
			limit = pToken.Size
		}

		if pToken.Token != "" {
			var parsed Pagination
			err := json.Unmarshal([]byte(pToken.Token), &parsed)
			if err != nil {
				return 0, 0, "", err
			}
			offset = parsed.Offset
			pagingRequestId = parsed.PagingRequestId
		}
	}
	return offset, limit, pagingRequestId, nil
}

// GetNextToken given a limit and offset that were used to fetch _this_ page of
// data, and total number of resources, return the next pagination token as a
// string.
func GetNextToken(
	offset int,
	limit int,
	total int,
) string {
	nextOffset := offset + limit
	if nextOffset >= total {
		return ""
	}

	bytes, err := json.Marshal(
		Pagination{
			Offset: nextOffset,
		},
	)
	if err != nil {
		return ""
	}

	return string(bytes)
}
