package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
)

type staffLeavePaginationTestService struct {
	StaffLeaveManagementService
	page, size int
	status string
	staffID int64
}
func (s *staffLeavePaginationTestService) ListRequestsPage(_ context.Context, _ int64, _ int64, status string, staffID int64, _, _, _ string, page, size int) (repository.StaffLeaveRequestPage, error) {
	s.page,s.size,s.status,s.staffID=page,size,status,staffID
	return repository.StaffLeaveRequestPage{Items:[]repository.StaffLeaveRequest{{ID:91}},Total:121},nil
}

type staffSubstitutionPaginationTestService struct {
	StaffSubstitutionManagementService
	page,size int
	staffID int64
}
func (s *staffSubstitutionPaginationTestService) ListPage(_ context.Context,_,_ int64,_,_ string,staffID int64,page,size int)(repository.StaffSubstitutionPage,error){
	s.page,s.size,s.staffID=page,size,staffID
	return repository.StaffSubstitutionPage{Items:[]repository.StaffSubstitution{{ID:92}},Total:101},nil
}

func staffPaginationAuth(c *gin.Context){
	c.Set(middleware.ContextPrincipalIDKey,int64(100))
	c.Set(middleware.ContextSchoolIDKey,int64(10))
	c.Next()
}

func TestStaffLeavePaginationHandlerForwardsFiltersAndReturnsTotals(t *testing.T){
	gin.SetMode(gin.TestMode)
	service:=&staffLeavePaginationTestService{}
	router:=gin.New()
	router.GET("/leave",staffPaginationAuth,NewStaffLeaveHandler(service).ListRequests)
	recorder:=httptest.NewRecorder()
	router.ServeHTTP(recorder,httptest.NewRequest(http.MethodGet,"/leave?status=approved&staff_principal_id=501&page=3&page_size=25",nil))
	if recorder.Code!=http.StatusOK{t.Fatalf("status=%d body=%s",recorder.Code,recorder.Body.String())}
	if service.page!=3||service.size!=25||service.status!="approved"||service.staffID!=501{t.Fatalf("pagination/filter forwarding mismatch: %#v",service)}
	var body struct{Data []repository.StaffLeaveRequest `json:"data"`;Pagination struct{Page int `json:"page"`;PageSize int `json:"page_size"`;Total int64 `json:"total"`} `json:"pagination"`}
	if err:=json.Unmarshal(recorder.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
	if len(body.Data)!=1||body.Data[0].ID!=91||body.Pagination.Page!=3||body.Pagination.PageSize!=25||body.Pagination.Total!=121{t.Fatalf("unexpected paginated leave response: %#v",body)}
}

func TestStaffSubstitutionPaginationHandlerForwardsFiltersAndReturnsTotals(t *testing.T){
	gin.SetMode(gin.TestMode)
	service:=&staffSubstitutionPaginationTestService{}
	router:=gin.New()
	router.GET("/substitutions",staffPaginationAuth,NewStaffSubstitutionHandler(service).List)
	recorder:=httptest.NewRecorder()
	router.ServeHTTP(recorder,httptest.NewRequest(http.MethodGet,"/substitutions?from_date=2026-11-01&to_date=2026-11-30&staff_principal_id=502&page=2&page_size=40",nil))
	if recorder.Code!=http.StatusOK{t.Fatalf("status=%d body=%s",recorder.Code,recorder.Body.String())}
	if service.page!=2||service.size!=40||service.staffID!=502{t.Fatalf("pagination/filter forwarding mismatch: %#v",service)}
	var body struct{Items []repository.StaffSubstitution `json:"substitutions"`;Pagination struct{Page int `json:"page"`;PageSize int `json:"page_size"`;Total int64 `json:"total"`} `json:"pagination"`}
	if err:=json.Unmarshal(recorder.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
	if len(body.Items)!=1||body.Items[0].ID!=92||body.Pagination.Page!=2||body.Pagination.PageSize!=40||body.Pagination.Total!=101{t.Fatalf("unexpected paginated substitution response: %#v",body)}
}

func TestStaffPaginationHandlersRejectInvalidPageParameters(t *testing.T){
	gin.SetMode(gin.TestMode)
	tests:=[]struct{name,query string}{
		{"zero page","?page=0"},
		{"negative page","?page=-1"},
		{"non-numeric page","?page=abc"},
		{"zero page size","?page_size=0"},
		{"page size above maximum","?page_size=101"},
		{"non-numeric page size","?page_size=abc"},
	}
	for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){
		router:=gin.New()
		router.GET("/leave",staffPaginationAuth,NewStaffLeaveHandler(&staffLeavePaginationTestService{}).ListRequests)
		recorder:=httptest.NewRecorder()
		router.ServeHTTP(recorder,httptest.NewRequest(http.MethodGet,"/leave"+tt.query,nil))
		if recorder.Code!=http.StatusBadRequest{t.Fatalf("status=%d want=%d body=%s",recorder.Code,http.StatusBadRequest,recorder.Body.String())}
	})}
}
