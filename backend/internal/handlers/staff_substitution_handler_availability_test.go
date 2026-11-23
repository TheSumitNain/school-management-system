package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
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


type nilStaffSubstitutionListService struct { StaffSubstitutionManagementService }
func (nilStaffSubstitutionListService) List(context.Context,int64,int64,string,string,int64)([]repository.StaffSubstitution,error){return nil,nil}

func TestStaffSubstitutionListReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r:=gin.New()
	r.GET("/substitutions",func(c *gin.Context){c.Set(middleware.ContextPrincipalIDKey,int64(100));c.Set(middleware.ContextSchoolIDKey,int64(10));c.Next()},NewStaffSubstitutionHandler(nilStaffSubstitutionListService{}).List)
	rec:=httptest.NewRecorder();r.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,"/substitutions",nil))
	if rec.Code!=http.StatusOK || !strings.Contains(rec.Body.String(),`"substitutions":[]`) {t.Fatalf("status=%d body=%s",rec.Code,rec.Body.String())}
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

type staffSubstitutionCancelTestService struct {
	StaffSubstitutionManagementService
	err error
	actorID, schoolID, substitutionID int64
}

func (s *staffSubstitutionCancelTestService) Cancel(_ context.Context, actorID, schoolID, substitutionID int64) error {
	s.actorID, s.schoolID, s.substitutionID = actorID, schoolID, substitutionID
	return s.err
}

func TestStaffSubstitutionCancelHandlerMapsServiceResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		id string
		err error
		wantStatus int
		wantMessage string
	}{
		{name: "success", id: "200", wantStatus: http.StatusOK, wantMessage: "Substitution cancelled; history has been retained"},
		{name: "not found or already cancelled", id: "200", err: repository.ErrStaffSubstitutionNotFound, wantStatus: http.StatusNotFound, wantMessage: "Staff substitution not found or already cancelled"},
		{name: "access denied", id: "200", err: repository.ErrStaffSubstitutionAccessDenied, wantStatus: http.StatusForbidden, wantMessage: "You are not authorized to manage staff substitutions"},
		{name: "invalid identifier", id: "0", wantStatus: http.StatusBadRequest, wantMessage: "Invalid substitution ID"},
		{name: "invalid identifier text", id: "abc", wantStatus: http.StatusBadRequest, wantMessage: "Invalid substitution ID"},
		{name: "unexpected service failure", id: "200", err: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError, wantMessage: "Unable to process staff substitution"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &staffSubstitutionCancelTestService{err: tt.err}
			router := gin.New()
			router.POST("/staff/substitutions/:substitutionID/cancel", func(c *gin.Context) {
				c.Set(middleware.ContextPrincipalIDKey, int64(100))
				c.Set(middleware.ContextSchoolIDKey, int64(10))
				c.Next()
			}, NewStaffSubstitutionHandler(service).Cancel)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/staff/substitutions/"+tt.id+"/cancel", nil))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tt.wantMessage) {
				t.Fatalf("response %q does not contain %q", recorder.Body.String(), tt.wantMessage)
			}
			if tt.wantStatus == http.StatusOK {
				if service.actorID != 100 || service.schoolID != 10 || service.substitutionID != 200 {
					t.Fatalf("service received wrong scope: actor=%d school=%d substitution=%d", service.actorID, service.schoolID, service.substitutionID)
				}
			} else if tt.wantStatus == http.StatusBadRequest && service.substitutionID != 0 {
				t.Fatalf("invalid identifier reached service: %+v", service)
			}
		})
	}
}

