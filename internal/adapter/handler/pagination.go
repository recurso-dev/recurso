package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams represents parsed pagination parameters
type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Limit   int `json:"-"`
	Offset  int `json:"-"`
}

// ParsePagination extracts pagination params from query string.
// Defaults: page=1, per_page=50, max per_page=250
func ParsePagination(c *gin.Context) PaginationParams {
	page := 1
	perPage := 50

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			perPage = v
		}
	}

	// Also support "limit" and "offset" directly
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			perPage = v
		}
	}

	// Cap at 250
	if perPage > 250 {
		perPage = 250
	}

	offset := (page - 1) * perPage

	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
		Limit:   perPage,
		Offset:  offset,
	}
}

// clampLimitOffset bounds ad-hoc limit/offset query params that predate
// ParsePagination: limit falls back to def when non-positive and is capped
// at max; offset is floored at 0. Negative values otherwise reach Postgres
// as invalid LIMIT/OFFSET (or force unbounded scans).
func clampLimitOffset(limit, offset, def, max int) (int, int) {
	if limit <= 0 || limit > max {
		limit = def
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
