package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	result, err := h.churnService.GetCustomerScore(ctx, customerID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *ChurnHandler) GetHighRiskCustomers(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	query := `SELECT id, tenant_id, customer_id, previous_score, new_score, threshold, alert_type, acknowledged, created_at
		FROM churn_alerts WHERE tenant_id = $1 AND acknowledged = FALSE ORDER BY created_at DESC LIMIT 100`

	rows, err := h.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	defer func() { _ = rows.Close() }()

	var alerts []churnAlert
	for rows.Next() {
		var a churnAlert
		if err := rows.Scan(&a.ID, &a.TenantID, &a.CustomerID, &a.PreviousScore, &a.NewScore,
			&a.Threshold, &a.AlertType, &a.Acknowledged, &a.CreatedAt); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
			return
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid alert id")
		return
	}

	query := `UPDATE churn_alerts SET acknowledged = TRUE WHERE id = $1 AND tenant_id = $2`
	result, err := h.db.ExecContext(c.Request.Context(), query, alertID, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		respondError(c, http.StatusNotFound, codeNotFound, "alert not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}
