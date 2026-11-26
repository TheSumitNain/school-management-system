package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseStaffPageBoundsComputedOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		page, size, ok := parseStaffPage(c)
		if !ok {
			c.Status(http.StatusBadRequest)
			return
		}
		if page != 10001 || size != 100 {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	valid := httptest.NewRecorder()
	router.ServeHTTP(valid, httptest.NewRequest(http.MethodGet, "/?page=10001&page_size=100", nil))
	if valid.Code != http.StatusNoContent {
		t.Fatalf("maximum supported page offset status=%d want=%d", valid.Code, http.StatusNoContent)
	}

	tooLarge := httptest.NewRecorder()
	router.ServeHTTP(tooLarge, httptest.NewRequest(http.MethodGet, "/?page=10002&page_size=100", nil))
	if tooLarge.Code != http.StatusBadRequest {
		t.Fatalf("oversized page offset status=%d want=%d", tooLarge.Code, http.StatusBadRequest)
	}
}
