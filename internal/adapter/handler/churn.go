package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/service"
)

type ChurnHandler struct {
	churnService *service.ChurnService
	db           *sql.DB
}

func NewChurnHandler(churnService *service.ChurnService, db *sql.DB) *ChurnHandler {
	return &ChurnHandler{
		churnService: churnService,
		db:           db,
	}
}

func (h *ChurnHandler) GetCustomerChurn(c *gin.Context) {
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	result, err := h.churnService.GetCustomerScore(c.Request.Context(), customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *ChurnHandler) GetHighRiskCustomers(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	threshold := 70
	if t := c.Query("threshold"); t != "" {
		if v, err := strconv.Atoi(t); err == nil && v > 0 {
			threshold = v
		}
	}

	results, err := h.churnService.GetHighRiskCustomers(c.Request.Context(), tenantID, threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if results == nil {
		results = []*service.ChurnScoreResult{}
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

type churnAlert struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	PreviousScore int       `json:"previous_score"`
	NewScore      int       `json:"new_score"`
	Threshold     int       `json:"threshold"`
	AlertType     string    `json:"alert_type"`
	Acknowledged  bool      `json:"acknowledged"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *ChurnHandler) GetAlerts(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	query := `SELECT id, tenant_id, customer_id, previous_score, new_score, threshold, alert_type, acknowledged, created_at
		FROM churn_alerts WHERE tenant_id = $1 AND acknowledged = FALSE ORDER BY created_at DESC LIMIT 100`

	rows, err := h.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = rows.Close() }()

	var alerts []churnAlert
	for rows.Next() {
		var a churnAlert
		if err := rows.Scan(&a.ID, &a.TenantID, &a.CustomerID, &a.PreviousScore, &a.NewScore,
			&a.Threshold, &a.AlertType, &a.Acknowledged, &a.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if alerts == nil {
		alerts = []churnAlert{}
	}

	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

func (h *ChurnHandler) AcknowledgeAlert(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	query := `UPDATE churn_alerts SET acknowledged = TRUE WHERE id = $1 AND tenant_id = $2`
	result, err := h.db.ExecContext(c.Request.Context(), query, alertID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}
