package handler

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type wompiWebhookVerifier interface {
	VerifyWebhookChecksum(parsed map[string]any, signature string) bool
}

// HandleWompi processes asynchronous webhook notifications from Wompi.
// Events include "transaction.updated" and "refund.updated".
func (h *WebhookHandler) HandleWompi(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read wompi webhook body", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "failed to read body")
		return
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		h.logger.Error("failed to unmarshal wompi webhook body", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid JSON payload")
		return
	}

	// 1. Resolve signing secret / gateway connection (BYO vs Env)
	connID := c.Param("connID")
	var webhookSecret string
	if connID != "" {
		sec, ok := h.webhookSecretFor(c, domain.GatewayWompi, "")
		if !ok {
			return
		}
		webhookSecret = sec
	} else {
		webhookSecret = h.wompiEventsSecret
	}

	// Extract or construct verifier
	signature := c.GetHeader("X-Event-Checksum")
	var verified bool
	if webhookSecret != "" {
		verified = verifyWompiChecksumWithSecret(parsed, signature, webhookSecret)
	} else if wgw, ok := h.gateway.(wompiWebhookVerifier); ok {
		verified = wgw.VerifyWebhookChecksum(parsed, signature)
	}

	if !verified {
		h.logger.Warn("wompi webhook signature verification failed", "ip", c.ClientIP())
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid webhook signature")
		return
	}

	event, _ := parsed["event"].(string)
	data, _ := parsed["data"].(map[string]any)
	h.logger.Info("wompi webhook received", "event", event)

	ctx := c.Request.Context()

	// 2. Handle transaction.updated
	switch event {
	case "transaction.updated":
		txData, _ := data["transaction"].(map[string]any)
		if txData == nil {
			c.JSON(http.StatusOK, gin.H{"status": "ignored (empty transaction)"})
			return
		}

		txID, _ := txData["id"].(string)
		status, _ := txData["status"].(string)
		ref, _ := txData["reference"].(string)

		if h.alreadyProcessed(c, "wompi", txID) {
			return
		}

		if status == "APPROVED" {
			// Extract invoice ID from reference: inv_{invoice_id}_{timestamp}
			invoiceIDStr := ref
			parts := strings.Split(ref, "_")
			if len(parts) >= 2 {
				invoiceIDStr = parts[1]
			}

			invoiceID, err := uuid.Parse(invoiceIDStr)
			if err != nil {
				h.logger.Warn("invalid invoice ID in wompi reference", "ref", ref)
				c.JSON(http.StatusOK, gin.H{"status": "ack"})
				return
			}

			inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
			if err != nil || inv == nil {
				h.logger.Warn("wompi transaction references unknown invoice", "invoice_id", invoiceIDStr)
				c.JSON(http.StatusOK, gin.H{"status": "ack"})
				return
			}

			if !invoiceBelongsToWebhookConn(ctx, inv) {
				h.logger.Warn("BYO wompi webhook referenced another tenant's invoice", "invoice_id", inv.ID)
				c.JSON(http.StatusOK, gin.H{"status": "ack"})
				return
			}

			ctxWithTenant := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
			transitioned, err := h.subService.MarkInvoicePaid(ctxWithTenant, invoiceID)
			if err != nil {
				h.logger.Error("failed to mark invoice paid via wompi webhook", "invoice_id", invoiceID, "error", err)
				respondInternalError(c, err)
				return
			}

			if h.invoiceRepo != nil && txID != "" {
				_ = h.invoiceRepo.SetGatewayPaymentID(ctx, inv.TenantID, invoiceID, txID)
			}

			if transitioned {
				h.recordDunningSuccess(ctx, invoiceID)
			}
		}

		h.markProcessed(ctx, "wompi", txID, event)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func verifyWompiChecksumWithSecret(parsed map[string]any, signature string, eventsSecret string) bool {
	if eventsSecret == "" {
		return false
	}

	sigObj, ok := parsed["signature"].(map[string]any)
	if !ok {
		return false
	}
	props, ok := sigObj["properties"].([]any)
	if !ok {
		return false
	}

	dataObj, _ := parsed["data"].(map[string]any)
	var concat strings.Builder
	for _, p := range props {
		path, ok := p.(string)
		if !ok {
			continue
		}
		val := resolveWompiProperty(dataObj, path)
		concat.WriteString(fmt.Sprint(val))
	}

	var tsStr string
	if ts, ok := parsed["timestamp"].(float64); ok {
		tsStr = strconv.FormatInt(int64(ts), 10)
	} else if ts, ok := parsed["timestamp"].(int64); ok {
		tsStr = strconv.FormatInt(ts, 10)
	} else {
		tsStr = fmt.Sprint(parsed["timestamp"])
	}
	concat.WriteString(tsStr)
	concat.WriteString(eventsSecret)

	expected := sha256.Sum256([]byte(concat.String()))
	expectedHex := hex.EncodeToString(expected[:])

	if signature == "" {
		if c, ok := sigObj["checksum"].(string); ok {
			signature = c
		}
	}

	return subtle.ConstantTimeCompare([]byte(strings.ToLower(expectedHex)), []byte(strings.ToLower(signature))) == 1
}

func resolveWompiProperty(m map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		if currMap, ok := current.(map[string]any); ok {
			current = currMap[part]
		} else {
			return nil
		}
	}
	return current
}
