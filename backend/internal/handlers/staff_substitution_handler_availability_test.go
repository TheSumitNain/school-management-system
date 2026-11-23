package handlers

import (
	"context"\n\t"net/http"
	"net/http/httptest"\n\t"strings"
	"testing"

	"school-management-system/internal/middleware"\n\t"school-management-system/internal/repository"
	"github.com/gin-gonic/gin"
)

func TestStaffSubstitutionHandlersReturnUnavailableWithoutService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	methods := []struct {
		name string
		handler func(*StaffSubstitutionHandler, *gin.Context)
	}{
		{"list substitutions", (*StaffSubstitutionHandler).List},
		{"create substitution", (*StaffSubstitutionHandler).Create},
		{"cancel substitution", (*StaffSubstitutionHandler).Cancel},
	}
	for _, method := range methods {
		for _, receiver := range []struct {
			name string
			handler *StaffSubstitutionHandler
		}{
			{"nil receiver", nil},
			{"nil service", NewStaffSubstitutionHandler(nil)},
		} {
			t.Run(method.name+" "+receiver.name, func(t *testing.T) {
				router := gin.New()
				router.Any("/staff/:substitutionID", func(c *gin.Context) {
					c.Set(middleware.ContextPrincipalIDKey, int64(100))
					c.Set(middleware.ContextSchoolIDKey, int64(10))
					method.handler(receiver.handler, c)
				})
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/staff/200", nil))
				if recorder.Code != http.StatusServiceUnavailable {
					t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
				}
			})
		}
	}
}


type nilStaffSubstitutionWriteResultService struct { StaffSubstitutionManagementService }
func (nilStaffSubstitutionWriteResultService) Create(context.Context,int64,int64,int64,int64,int64,string,*int64,string)(*repository.StaffSubstitution,error){return nil,nil}

func TestStaffSubstitutionCreateRejectsNilSuccessfulResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r:=gin.New()
	r.POST("/staff/substitutions",func(c *gin.Context){
		c.Set(middleware.ContextPrincipalIDKey,int64(100));c.Set(middleware.ContextSchoolIDKey,int64(10));c.Next()
	},NewStaffSubstitutionHandler(nilStaffSubstitutionWriteResultService{}).Create)
	req:=httptest.NewRequest(http.MethodPost,"/staff/substitutions",strings.NewReader(`{}`));req.Header.Set("Content-Type","application/json")
	rec:=httptest.NewRecorder();r.ServeHTTP(rec,req)
	if rec.Code!=http.StatusInternalServerError{t.Fatalf("nil successful service result must return 500: status=%d body=%s",rec.Code,rec.Body.String())}
}
