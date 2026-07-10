package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// waitlistStore persists signups. Satisfied by *db.WaitlistRepository.
type waitlistStore interface {
	Add(ctx context.Context, email, name, company, source string) (bool, error)
}

// WaitlistHandler serves the public Recurso Cloud waitlist form (ENG-12).
type WaitlistHandler struct {
	store waitlistStore
}

func NewWaitlistHandler(store waitlistStore) *WaitlistHandler {
	return &WaitlistHandler{store: store}
}

type joinWaitlistRequest struct {
	Email   string `json:"email" binding:"required,email,max=320"`
	Name    string `json:"name" binding:"max=200"`
	Company string `json:"company" binding:"max=200"`
	Source  string `json:"source" binding:"max=100"`
	// Website is a honeypot: hidden on the real form, so any value means a
	// bot. We report success without storing anything.
	Website string `json:"website"`
}

// Join handles POST /waitlist. Public and rate-limited. The response is the
// same for new signups, repeats, and honeypot hits — nothing to enumerate.
func (h *WaitlistHandler) Join(c *gin.Context) {
	var req joinWaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "a valid email is required")
		return
	}

	if req.Website == "" {
		if _, err := h.store.Add(c.Request.Context(), req.Email, req.Name, req.Company, req.Source); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "could not join the waitlist right now")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message": "You're on the list — we'll be in touch.",
	}})
}
