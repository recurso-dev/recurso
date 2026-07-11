package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// TeamHandler serves the tenant-scoped team-management endpoints under /v1/users.
// It relies on the dual-auth middleware having set tenant_id (always) and, for
// dashboard users, user_id + user_role.
type TeamHandler struct {
	auth *service.AuthService
}

func NewTeamHandler(auth *service.AuthService) *TeamHandler {
	return &TeamHandler{auth: auth}
}

func (h *TeamHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := c.MustGet("tenant_id").(uuid.UUID)
	return id, ok
}

// requireManager gates write operations to owner/admin. Requests authenticated
// by API key (a machine, no user context) are allowed — they already have full
// tenant access on every other v1 endpoint. Members are rejected with 403.
func (h *TeamHandler) requireManager(c *gin.Context) bool {
	role, hasUser := middleware.GetUserRole(c)
	if !hasUser {
		return true // machine (API key) caller
	}
	if domain.Role(role).CanManageTeam() {
		return true
	}
	respondError(c, http.StatusForbidden, codeForbidden, "requires owner or admin role")
	return false
}

// mapTeamError translates domain errors into HTTP responses.
func mapTeamError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, "user not found")
	case errors.Is(err, domain.ErrDuplicateEmail):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	case errors.Is(err, domain.ErrLastOwner), errors.Is(err, domain.ErrSelfLockout):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	case errors.Is(err, domain.ErrWeakPassword), errors.Is(err, domain.ErrInvalidRole):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}

// GET /v1/users
func (h *TeamHandler) ListUsers(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	users, err := h.auth.ListUsers(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	views := make([]userView, 0, len(users))
	for _, u := range users {
		views = append(views, toUserView(u))
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// POST /v1/users
func (h *TeamHandler) CreateUser(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	user, err := h.auth.CreateUser(c.Request.Context(), tenantID, req.Email, req.Name, domain.Role(req.Role), req.Password)
	if err != nil {
		mapTeamError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toUserView(user)})
}

type updateUserRequest struct {
	Role string `json:"role" binding:"required"`
}

// PATCH /v1/users/:id
func (h *TeamHandler) UpdateUser(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid user id")
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	user, err := h.auth.UpdateUserRole(c.Request.Context(), tenantID, targetID, domain.Role(req.Role))
	if err != nil {
		mapTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toUserView(user)})
}

// DELETE /v1/users/:id
func (h *TeamHandler) DeleteUser(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid user id")
		return
	}
	actingUserID := middleware.GetUserID(c) // uuid.Nil for API-key callers
	if err := h.auth.DeleteUser(c.Request.Context(), tenantID, actingUserID, targetID); err != nil {
		mapTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user removed"})
}
