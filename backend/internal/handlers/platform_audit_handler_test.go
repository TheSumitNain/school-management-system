package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
)

type platformAuditServiceFake struct {
	actorID int64
	limit int
	offset int
}
func (f *platformAuditServiceFake) List(_ context.Context, actorID int64, limit, offset int) (*repository.PlatformAuditPage, error) {
	f.actorID, f.limit, f.offset = actorID, limit, offset
	return &repository.PlatformAuditPage{Events: []repository.PlatformAuditEvent{}, Total: 0}, nil
}

func TestPlatformAuditHandlerParsesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &platformAuditServiceFake{}
	handler := NewPlatformAuditHandler(service)
	router := gin.New()
	router.GET("/platform/audit-events", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(42))
		handler.List(c)
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/platform/audit-events?limit=25&offset=50", nil)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if service.actorID != 42 || service.limit != 25 || service.offset != 50 {
		t.Fatalf("service args = actor %d limit %d offset %d", service.actorID, service.limit, service.offset)
	}
}

func TestPlatformAuditHandlerRejectsInvalidPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPlatformAuditHandler(&platformAuditServiceFake{})
	router := gin.New()
	router.GET("/platform/audit-events", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(42))
		handler.List(c)
	})
	for _, query := range []string{"limit=0", "limit=101", "limit=abc", "offset=-1", "offset=abc"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/platform/audit-events?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Errorf("query %q status = %d, want %d", query, response.Code, http.StatusBadRequest)
		}
	}
}
