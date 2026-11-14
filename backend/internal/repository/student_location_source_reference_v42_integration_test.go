package repository

import (
	"context"
	"testing"
	"time"
)

// TestStudentLocationSourceReferencesV42PostgresIntegration verifies that a
// canonical location can retain its original IGOD code while receiving
// versioned LGD identifiers without overwriting source provenance.
func TestStudentLocationSourceReferencesV42PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V42 integration test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var seeded int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM student_location_source_references
		 WHERE source_system = 'IGOD' AND is_current = TRUE
	`).Scan(&seeded); err != nil {
		t.Fatalf("count backfilled IGOD source references: %v", err)
	}
	if seeded != 36 {
		t.Fatalf("V42 backfilled %d IGOD source references, want 36", seeded)
	}

	var haryanaID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM student_locations
		 WHERE school_id IS NULL AND source_system = 'IGOD'
		   AND source_code = 'HR' AND level = 'state'
	`).Scan(&haryanaID); err != nil {
		t.Fatalf("find canonical Haryana State row: %v", err)
	}

	var lgdReferenceID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_location_source_references
			(location_id, source_system, source_code, source_name, snapshot_date)
		VALUES ($1, 'LGD_STATE', '6', 'Haryana', DATE '2026-11-14')
		RETURNING id
	`, haryanaID).Scan(&lgdReferenceID); err != nil {
		t.Fatalf("add LGD state identifier alongside IGOD identifier: %v", err)
	}

	var currentSources int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_location_source_references
		 WHERE location_id = $1 AND is_current = TRUE
	`, haryanaID).Scan(&currentSources); err != nil {
		t.Fatalf("count current Haryana source identifiers: %v", err)
	}
	if currentSources != 2 {
		t.Fatalf("Haryana has %d current source identifiers, want IGOD + LGD", currentSources)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE student_location_source_references
		   SET is_current = FALSE, superseded_at = NOW()
		 WHERE id = $1
	`, lgdReferenceID); err != nil {
		t.Fatalf("supersede old LGD source identifier: %v", err)
	}
	var replacementID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_location_source_references
			(location_id, source_system, source_code, source_name, snapshot_date)
		VALUES ($1, 'LGD_STATE', '06', 'Haryana', DATE '2026-11-15')
		RETURNING id
	`, haryanaID).Scan(&replacementID); err != nil {
		t.Fatalf("insert replacement LGD source identifier while retaining history: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE student_location_source_references
		   SET source_code = '999'
		 WHERE id = $1
	`, replacementID); err == nil {
		t.Fatal("source identifier update succeeded; source snapshots must be immutable")
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM student_location_source_references WHERE id = $1
	`, replacementID); err == nil {
		t.Fatal("source reference delete succeeded; source history must be append-only")
	}

	var historyCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_location_source_references
		 WHERE location_id = $1 AND source_system = 'LGD_STATE'
	`, haryanaID).Scan(&historyCount); err != nil {
		t.Fatalf("count LGD source history: %v", err)
	}
	if historyCount != 2 {
		t.Fatalf("LGD source history has %d rows, want 2", historyCount)
	}
}
