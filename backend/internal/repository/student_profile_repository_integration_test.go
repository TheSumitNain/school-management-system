package repository

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestStudentProfileRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var schoolID, actorID int64
	if err := pool.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &actorID); err != nil {
		t.Fatalf("load school admin fixture: %v", err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	repo := NewAcademicRepository(pool)

	student, err := repo.CreateStudent(ctx, actorID, schoolID, " ADM-"+suffix+" ", "  Student One  ")
	if err != nil { t.Fatalf("create student admission: %v", err) }
	if student.AdmissionNumber != "ADM-"+suffix || student.FullName != "Student One" || student.Status != "active" {
		t.Fatalf("unexpected created student: %+v", student)
	}
	if _, err := repo.CreateStudent(ctx, actorID, schoolID, "adm-"+suffix, "Duplicate Admission"); !errors.Is(err, ErrAcademicConflict) {
		t.Fatalf("duplicate normalized admission error = %v, want conflict", err)
	}
	got, err := repo.GetStudent(ctx, actorID, schoolID, student.ID)
	if err != nil || got.ID != student.ID { t.Fatalf("get student: student=%+v err=%v", got, err) }

	updated, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Student One Updated")
	if err != nil || updated.FullName != "Student One Updated" {
		t.Fatalf("update student: student=%+v err=%v", updated, err)
	}
	items, total, err := repo.ListStudents(ctx, actorID, schoolID, "Updated", "active", 10, 0)
	if err != nil || total < 1 { t.Fatalf("search students: total=%d err=%v", total, err) }
	found := false
	for _, item := range items { if item.ID == student.ID { found = true } }
	if !found { t.Fatalf("student %d missing from school-scoped search: %+v", student.ID, items) }

	archived, err := repo.ArchiveStudent(ctx, actorID, schoolID, student.ID)
	if err != nil || archived.Status != "archived" { t.Fatalf("archive student: student=%+v err=%v", archived, err) }
	if _, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Should Not Update"); !errors.Is(err, ErrAcademicNotFound) {
		t.Fatalf("update archived student error = %v, want not found", err)
	}

	var otherSchoolID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO schools (name, code, status) VALUES ($1, $2, 'active') RETURNING id
	`, "Student Profile Isolation "+suffix, "SPI-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create isolation school: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM audit_events WHERE school_id = $1 AND resource_type = 'student' AND resource_id = $2`, schoolID, strconv.FormatInt(student.ID, 10))
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM students WHERE school_id = $1 AND id = $2`, schoolID, student.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM schools WHERE id = $1`, otherSchoolID)
	})
	if _, err := repo.GetStudent(ctx, actorID, otherSchoolID, student.ID); !errors.Is(err, ErrAcademicNotFound) {
		t.Fatalf("cross-school student lookup error = %v, want not found", err)
	}
}
