package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
)

type TenantRegionLookup interface {
	GetTenantRegion(tenantID uuid.UUID) string
}

// RegionMiddleware sets the data region in the request context based on the tenant's configuration.
func RegionMiddleware(lookup TenantRegionLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := c.Get("tenant_id")
		if !ok {
			c.Next()
			return
		}

		tid, ok := tenantID.(uuid.UUID)
		if !ok {
			c.Next()
			return
		}

		region := lookup.GetTenantRegion(tid)
		if region != "" {
			ctx := db.ContextWithRegion(c.Request.Context(), region)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}
