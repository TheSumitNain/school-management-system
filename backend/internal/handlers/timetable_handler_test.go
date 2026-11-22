package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

type timetableListEntriesTestService struct {
	gotActor, gotSchool, gotYear, gotSection, gotTeacher int64
}

func (s *timetableListEntriesTestService) CurrentYear(context.Context, int64, int64) (*repository.AcademicYear, error) {
	return nil, nil
}
func (s *timetableListEntriesTestService) ListPeriods(context.Context, int64, int64, int64) ([]repository.TimetablePeriod, error) {
	return nil, nil
}
func (s *timetableListEntriesTestService) SavePeriod(context.Context, int64, int64, int64, int64, int, string, string, string, bool) (*repository.TimetablePeriod, error) {
	return nil, nil
}
func (s *timetableListEntriesTestService) ListEntries(_ context.Context, actor, school, year, section, teacher int64) ([]repository.TimetableEntry, error) {
	s.gotActor, s.gotSchool, s.gotYear, s.gotSection, s.gotTeacher = actor, school, year, section, teacher
	return []repository.TimetableEntry{}, nil
}
func (s *timetableListEntriesTestService) SaveEntry(context.Context, int64, int64, int64, int64, int64, int64, int64, int64, int64, int, *string) (*repository.TimetableEntry, error) {
	return nil, nil
}
func (s *timetableListEntriesTestService) DeleteEntry(context.Context, int64, int64, int64, int64) error {
	return nil
}

func newTimetableListEntriesTestContext(query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/school/timetable/entries?"+query, nil)
	ctx.Set(middleware.ContextPrincipalIDKey, int64(41))
	ctx.Set(middleware.ContextSchoolIDKey, int64(7))
	return ctx, recorder
}

func TestTimetableListEntriesPassesTeacherScope(t *testing.T) {
	service := &timetableListEntriesTestService{}
	handler := NewTimetableHandler(service)
	ctx, recorder := newTimetableListEntriesTestContext("academic_year_id=12&section_id=8&teacher_principal_id=55")

	handler.ListEntries(ctx)

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if service.gotActor != 41 || service.gotSchool != 7 || service.gotYear != 12 || service.gotSection != 8 || service.gotTeacher != 55 {
		t.Fatalf("unexpected scoped arguments: actor=%d school=%d year=%d section=%d teacher=%d",
			service.gotActor, service.gotSchool, service.gotYear, service.gotSection, service.gotTeacher)
	}
}

func TestTimetableListEntriesRejectsInvalidTeacherScope(t *testing.T) {
	service := &timetableListEntriesTestService{}
	handler := NewTimetableHandler(service)
	ctx, recorder := newTimetableListEntriesTestContext("academic_year_id=12&teacher_principal_id=0")

	handler.ListEntries(ctx)

	if recorder.Code != 400 {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if service.gotYear != 0 || service.gotTeacher != 0 {
		t.Fatalf("service must not be called for invalid teacher filter: %+v", service)
	}
}
