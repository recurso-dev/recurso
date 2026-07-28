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

// parseLimitOffset reads ?limit= and ?offset= and clamps them via
// clampLimitOffset — the one-liner for list endpoints that take plain
// limit/offset (the house convention for bounding previously-unbounded
// lists: def=max=1000, a pure DoS-bound that leaves every realistic
// caller's results unchanged while capping runaway sets).
func parseLimitOffset(c *gin.Context, def, max int) (int, int) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	return clampLimitOffset(limit, offset, def, max)
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

// Tier-1 list endpoints (customers, subscriptions, plans) use page + limit
// query params. These constants centralize the shared default and cap.
const (
	// defaultPageLimit is returned when the caller omits limit. 50 (raised
	// from the historical 10) makes silent truncation less likely for API
	// callers who forget the param; every dashboard consumer already passes
	// an explicit limit, so the default is invisible to the UI.
	defaultPageLimit = 50
	// maxPageLimit bounds one page so a caller cannot request an unbounded
	// scan. 1000 matches the largest limit any current consumer requests.
	maxPageLimit = 1000
)

// parsePageLimit reads `page` (1-based) and `limit` query params and returns a
// SQL limit/offset. limit falls back to defaultPageLimit when absent/invalid
// and is CAPPED at maxPageLimit (a caller asking for more gets the cap, not a
// surprise downgrade). page < 1 is treated as page 1.
func parsePageLimit(c *gin.Context) (limit, offset int) {
	limit = defaultPageLimit
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	page := 1
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 1 {
		page = v
	}
	return limit, (page - 1) * limit
}
