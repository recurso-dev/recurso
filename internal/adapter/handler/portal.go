package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

type PortalHandler struct {
	custRepo            port.CustomerRepository
	invoiceRepo         port.InvoiceRepository
	subscriptionService *service.SubscriptionService
	invoiceService      *service.InvoiceService
	customerService     *service.CustomerService
}

// NewPortalHandler handles both the old HTML Dashboard and the new API
func NewPortalHandler(
	custRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
	subscriptionService *service.SubscriptionService,
	invoiceService *service.InvoiceService,
	customerService *service.CustomerService,
) *PortalHandler {
	return &PortalHandler{
		custRepo:            custRepo,
		invoiceRepo:         invoiceRepo,
		subscriptionService: subscriptionService,
		invoiceService:      invoiceService,
		customerService:     customerService,
	}
}

// ShowDashboard provides the original HTML-based customer experience
func (h *PortalHandler) ShowDashboard(c *gin.Context) {
	idStr := c.Param("customer_id")
	custID, err := uuid.Parse(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Customer ID")
		return
	}

	customer, err := h.custRepo.GetByIDPublic(c.Request.Context(), custID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching customer")
		return
	}
	if customer == nil {
		c.String(http.StatusNotFound, "Customer not found")
		return
	}

	// Fetch Invoices using GetByCustomerID (This is what the original did)
	invoices, err := h.invoiceRepo.GetByCustomerID(c.Request.Context(), custID)
	if err != nil {
		log.Printf("Error fetching invoices for portal dashboard: %v", err)
	}

	// View Model
	type InvoiceVM struct {
		ID            string
		InvoiceNumber string
		Currency      string
		DisplayAmount string
		Status        string
	}

	invVMs := []InvoiceVM{}
	for _, inv := range invoices {
		invVMs = append(invVMs, InvoiceVM{
			ID:            inv.ID.String(),
			InvoiceNumber: inv.InvoiceNumber,
			Currency:      inv.Currency,
			DisplayAmount: fmt.Sprintf("%.2f", float64(inv.Total)/100.0),
			Status:        string(inv.Status),
		})
	}

	data := gin.H{
		"CustomerName": customer.Name,
		"Invoices":     invVMs,
	}

	c.HTML(http.StatusOK, "portal_dashboard.html", data)
}

// GetPortalData serves the React JSON payload
func (h *PortalHandler) GetPortalData(c *gin.Context) {
	customerIDStr := c.Param("customer_id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid customer ID format")
		return
	}

	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid tenant ID format")
		return
	}

	ctx := c.Request.Context()

	// 1. Fetch Customer
	customer, err := h.customerService.GetCustomer(ctx, tenantID, customerID)
	if err != nil {
		log.Printf("Error fetching customer for portal: %v", err)
		respondError(c, http.StatusNotFound, codeNotFound, "Customer not found")
		return
	}

	// 2. Fetch Subscriptions
	filter := domain.SubscriptionFilter{CustomerID: customerID}
	subscriptions, err := h.subscriptionService.ListSubscriptions(ctx, tenantID, filter)
	if err != nil {
		log.Printf("Error fetching subscriptions for portal: %v", err)
		subscriptions = nil
	}

	// 3. Fetch Invoices directly from Repo since we have GetByCustomerID
	invoices, err := h.invoiceRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Printf("Error fetching invoices for portal: %v", err)
		invoices = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"customer": gin.H{
			"id":    customer.ID,
			"name":  customer.Name,
			"email": customer.Email,
		},
		"subscriptions": subscriptions,
		"invoices":      invoices,
	})
}
