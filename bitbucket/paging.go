package bitbucket

import (
	"context"
	"fmt"
	"net/http"
)

// maxPageLen is the largest page Bitbucket serves from most list endpoints.
const maxPageLen = 100

// pageQuery returns the pagelen and page query parameters for a list request.
// PerPage defaults to and is capped at maxPageLen. A start page is counted in
// pages of PerPage, so the page size only shrinks to the limit when listing
// from the first page.
func pageQuery(page, perPage, limit int) string {
	if perPage <= 0 || perPage > maxPageLen {
		perPage = maxPageLen
	}
	if limit > 0 && perPage > limit && page <= 1 {
		perPage = limit
	}
	q := fmt.Sprintf("pagelen=%d", perPage)
	if page > 1 {
		q += fmt.Sprintf("&page=%d", page)
	}
	return q
}

type bbPage[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
}

// listPages fetches u and every page after it by following next links,
// stopping once limit results have been collected when limit is positive.
func listPages[B, T any](ctx context.Context, rs *bitbucketRepoService, u string, limit int, convert func(B) T) ([]T, error) {
	var all []T
	for u != "" {
		var page bbPage[B]
		if err := rs.doJSON(ctx, http.MethodGet, u, nil, &page); err != nil {
			return nil, err
		}
		for _, v := range page.Values {
			all = append(all, convert(v))
		}
		if limit > 0 && len(all) >= limit {
			break
		}
		u = page.Next
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}
