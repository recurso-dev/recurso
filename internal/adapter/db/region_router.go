package db

import (
	"context"
	"database/sql"
	"sync"
)

type contextKey string

const regionContextKey contextKey = "data_region"

// RegionRouter routes database queries to the appropriate regional pool.
// If only one DB is configured, all queries go to the default pool.
type RegionRouter struct {
	defaultDB *sql.DB
	pools     map[string]*sql.DB
	mu        sync.RWMutex
}

func NewRegionRouter(defaultDB *sql.DB) *RegionRouter {
	return &RegionRouter{
		defaultDB: defaultDB,
		pools:     make(map[string]*sql.DB),
	}
}

// AddPool registers a regional database pool.
func (r *RegionRouter) AddPool(region string, db *sql.DB) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pools[region] = db
}

// GetDB returns the appropriate database pool for the region set in context.
// Falls back to the default pool if no region is set or no regional pool exists.
func (r *RegionRouter) GetDB(ctx context.Context) *sql.DB {
	region, ok := ctx.Value(regionContextKey).(string)
	if !ok || region == "" || region == "global" {
		return r.defaultDB
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if pool, exists := r.pools[region]; exists {
		return pool
	}

	return r.defaultDB
}

// ContextWithRegion sets the data region in the context.
func ContextWithRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, regionContextKey, region)
}

// RegionFromContext extracts the data region from context.
func RegionFromContext(ctx context.Context) string {
	region, _ := ctx.Value(regionContextKey).(string)
	return region
}
