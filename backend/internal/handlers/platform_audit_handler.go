package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"
)

type PlatformAuditService interface {
	List(context.Context, int64, int, int) (*repository.PlatformAuditPage, error)
}

type PlatformAuditHandler struct { service PlatformAuditService }

func NewPlatformAuditHandler(service PlatformAuditService) *PlatformAuditHandler {
	return &PlatformAuditHandler{service: service}
}

func (h *PlatformAuditHandler) List(c *gin.Context) {
	value, exists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, ok := value.(int64)
	if !exists || !ok || actorID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"Authentication required"})
		return
	}
	limit := 50
	offset := 0
	var err error
	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"limit must be between 1 and 100"})
			return
		}
	}
	if raw := c.Query("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"offset must be zero or greater"})
			return
		}
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Platform audit history is unavailable"})
		return
	}
	page, err := h.service.List(c.Request.Context(), actorID, limit, offset)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status":"success","events":page.Events,"total":page.Total,"limit":limit,"offset":offset})
	case errors.Is(err, services.ErrPlatformAuditForbidden):
		c.JSON(http.StatusForbidden, gin.H{"status":"error","message":"Platform audit access denied"})
	case errors.Is(err, services.ErrInvalidPlatformAuditQuery):
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid audit query"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status":"error","message":"Unable to load platform audit history"})
	}
}
