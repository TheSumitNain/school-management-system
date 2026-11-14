package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

type fakeGuardianManagementService struct {
	guardians []repository.Guardian
	guardian *repository.Guardian
	links []repository.StudentGuardian
	relatedStudents []repository.StudentRelatedStudent
	link *repository.StudentGuardian
	total int64
	err error
	actorID, schoolID, guardianID, studentID, linkID int64
	limit, offset int
	query, status, relationship, reason string
	mobile, email *string
	primary, emergency, history bool
	calls int
}

func (f *fakeGuardianManagementService) ListGuardians(_ context.Context, actor, school int64, query, status string, limit, offset int) ([]repository.Guardian, int64, error) {
	f.calls++; f.actorID, f.schoolID, f.query, f.status, f.limit, f.offset = actor, school, query, status, limit, offset
	return f.guardians, f.total, f.err
}
func (f *fakeGuardianManagementService) GetGuardian(_ context.Context, actor, school, id int64) (*repository.Guardian, error) {
	f.calls++; f.actorID, f.schoolID, f.guardianID = actor, school, id
	return f.guardian, f.err
}
func (f *fakeGuardianManagementService) SaveGuardian(_ context.Context, actor, school, id int64, name string, mobile, email, alternateMobile, occupation, education, workplace *string, status string) (*repository.Guardian, error) {
	f.calls++; f.actorID, f.schoolID, f.guardianID, f.status = actor, school, id, status
	f.query, f.mobile, f.email = name, mobile, email
	return f.guardian, f.err
}
func (f *fakeGuardianManagementService) ListStudentGuardians(_ context.Context, actor, school, student int64, history bool) ([]repository.StudentGuardian, error) {
	f.calls++; f.actorID, f.schoolID, f.studentID, f.history = actor, school, student, history
	return f.links, f.err
}
func (f *fakeGuardianManagementService) ListStudentRelatedStudents(_ context.Context, actor, school, student int64) ([]repository.StudentRelatedStudent, error) {
	f.calls++; f.actorID, f.schoolID, f.studentID = actor, school, student
	return f.relatedStudents, f.err
}
func (f *fakeGuardianManagementService) LinkStudentGuardian(_ context.Context, actor, school, student, guardian int64, relationship string, primary, emergency bool) (*repository.StudentGuardian, error) {
	f.calls++; f.actorID, f.schoolID, f.studentID, f.guardianID = actor, school, student, guardian
	f.relationship, f.primary, f.emergency = relationship, primary, emergency
	return f.link, f.err
}
func (f *fakeGuardianManagementService) RevokeStudentGuardianLink(_ context.Context, actor, school, link int64, reason string) error {
	f.calls++; f.actorID, f.schoolID, f.linkID, f.reason = actor, school, link, reason
	return f.err
}

func guardianTestAuth(c *gin.Context) {
	c.Set("auth_principal_id", int64(31))
	c.Set("auth_school_id", int64(8))
	c.Next()
}

func TestGuardianHandlerRequiresAuthenticationAndService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/guardians", NewGuardianHandler(&fakeGuardianManagementService{}).ListGuardians)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/guardians", nil))
	if rec.Code != http.StatusUnauthorized { t.Fatalf("missing auth status=%d body=%s", rec.Code, rec.Body.String()) }

	router = gin.New()
	router.GET("/guardians", guardianTestAuth, NewGuardianHandler(nil).ListGuardians)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/guardians", nil))
	if rec.Code != http.StatusServiceUnavailable { t.Fatalf("nil service status=%d body=%s", rec.Code, rec.Body.String()) }
}

func TestGuardianHandlerListPaginationAndScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeGuardianManagementService{total: 12, guardians: []repository.Guardian{{ID: 5, FullName: "Guardian"}}}
	router := gin.New()
	router.GET("/guardians", guardianTestAuth, NewGuardianHandler(service).ListGuardians)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/guardians?q=Ravi&status=active&limit=25&offset=5", nil))
	if rec.Code != http.StatusOK || service.actorID != 31 || service.schoolID != 8 ||
		service.query != "Ravi" || service.status != "active" || service.limit != 25 || service.offset != 5 ||
		!strings.Contains(rec.Body.String(), "\"total\":12") {
		t.Fatalf("list status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
	for _, target := range []string{"/guardians?limit=0", "/guardians?limit=101", "/guardians?limit=abc", "/guardians?offset=-1", "/guardians?offset=no"} {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusBadRequest { t.Fatalf("%s status=%d", target, rec.Code) }
	}
}

func TestGuardianHandlerSaveCreateAndUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeGuardianManagementService{guardian: &repository.Guardian{ID: 9, FullName: "A Parent"}}
	router := gin.New()
	handler := NewGuardianHandler(service)
	router.POST("/guardians", guardianTestAuth, handler.SaveGuardian)
	router.PATCH("/guardians/:guardianID", guardianTestAuth, handler.SaveGuardian)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/guardians", strings.NewReader(`{"full_name":"A Parent"}`)))
	if rec.Code != http.StatusCreated || service.guardianID != 0 || service.status != "active" {
		t.Fatalf("create status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/guardians/9", strings.NewReader(`{"full_name":"A Parent","status":"inactive"}`)))
	if rec.Code != http.StatusOK || service.guardianID != 9 || service.status != "inactive" {
		t.Fatalf("update status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
	for _, tc := range []struct{ path, body string }{
		{"/guardians", "{"}, {"/guardians/0", `{"full_name":"x","status":"active"}`},
		{"/guardians/9", `{"full_name":"x"}`},
	} {
		rec = httptest.NewRecorder()
		method := http.MethodPost
		if tc.path != "/guardians" { method = http.MethodPatch }
		router.ServeHTTP(rec, httptest.NewRequest(method, tc.path, strings.NewReader(tc.body)))
		if rec.Code != http.StatusBadRequest { t.Fatalf("%s status=%d body=%s", tc.path, rec.Code, rec.Body.String()) }
	}
}

func TestGuardianHandlerListRelatedStudents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeGuardianManagementService{relatedStudents: []repository.StudentRelatedStudent{{StudentID: 42, FullName: "Related Student", SharedGuardianNames: []string{"Parent One"}}}}
	router := gin.New()
	router.GET("/students/:studentID/related-students", guardianTestAuth, NewGuardianHandler(service).ListStudentRelatedStudents)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/students/15/related-students", nil))
	if rec.Code != http.StatusOK || service.actorID != 31 || service.schoolID != 8 || service.studentID != 15 ||
		!strings.Contains(rec.Body.String(), "Related Student") {
		t.Fatalf("related-students status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
}

func TestGuardianHandlerGetAndStudentRelationshipLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeGuardianManagementService{guardian: &repository.Guardian{ID: 9}, link: &repository.StudentGuardian{ID: 14}}
	router := gin.New()
	h := NewGuardianHandler(service)
	router.GET("/guardians/:guardianID", guardianTestAuth, h.GetGuardian)
	router.GET("/students/:studentID/guardians", guardianTestAuth, h.ListStudentGuardians)
	router.POST("/students/:studentID/guardians", guardianTestAuth, h.LinkStudentGuardian)
	router.POST("/links/:linkID/revoke", guardianTestAuth, h.RevokeStudentGuardianLink)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/guardians/9", nil))
	if rec.Code != http.StatusOK || service.guardianID != 9 { t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String()) }

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/students/22/guardians?include_history=true", nil))
	if rec.Code != http.StatusOK || service.studentID != 22 || !service.history { t.Fatalf("list links status=%d service=%+v", rec.Code, service) }

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/students/22/guardians", strings.NewReader(`{"guardian_id":9,"relationship_type":"mother","is_primary_contact":true,"is_emergency_contact":true}`)))
	if rec.Code != http.StatusCreated || service.studentID != 22 || service.guardianID != 9 ||
		service.relationship != "mother" || !service.primary || !service.emergency {
		t.Fatalf("link status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/links/14/revoke", strings.NewReader(`{"reason":"relationship ended"}`)))
	if rec.Code != http.StatusOK || service.linkID != 14 || service.reason != "relationship ended" {
		t.Fatalf("revoke status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
}

func TestGuardianHandlerMapsRepositoryErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct{ err error; want int }{
		{repository.ErrGuardianAccessDenied, http.StatusForbidden},
		{repository.ErrGuardianNotFound, http.StatusNotFound},
		{repository.ErrGuardianConflict, http.StatusConflict},
		{repository.ErrInvalidGuardianInput, http.StatusBadRequest},
		{errors.New("private database detail"), http.StatusInternalServerError},
	} {
		service := &fakeGuardianManagementService{err: tc.err}
		router := gin.New()
		router.GET("/guardians", guardianTestAuth, NewGuardianHandler(service).ListGuardians)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/guardians", nil))
		if rec.Code != tc.want { t.Fatalf("error %v status=%d want=%d body=%s", tc.err, rec.Code, tc.want, rec.Body.String()) }
		if tc.want == http.StatusInternalServerError && strings.Contains(rec.Body.String(), "private database detail") {
			t.Fatalf("internal error leaked: %s", rec.Body.String())
		}
	}
}
