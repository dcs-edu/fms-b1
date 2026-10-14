// Package repository: internal/repository/parents.go
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/pkg"
)

func (r *Repository) LinkParent(ctx context.Context, parentID, admissionNo uuid.UUID) error {
	query :=
	`
		INSERT INTO parent_students (parent_id, admission_no)
		VALUES ($1, $2)
	`
	if _, err := r.DB.ExecContext(ctx, query, parentID, admissionNo); err != nil {
		return fmt.Errorf("link parent: %w", mapPgError(err))
	}
	return nil
}

func (r *Repository) IsParentOf(ctx context.Context, parentID, admissionNo uuid.UUID) (bool, error) {
	var linked bool
	query :=
	`
		SELECT EXISTS (
			SELECT 1 FROM parent_students WHERE parent_id = $1 AND admission_no = $2
		)
	`
	if err := r.DB.GetContext(ctx, &linked, query, parentID, admissionNo); err != nil {
		return false, fmt.Errorf("check parent link: %w", mapPgError(err))
	}
	return linked, nil
}

func (r *Repository) ListChildren(ctx context.Context, parentID uuid.UUID) ([]models.Std, error) {
	// Std has no grad_year/seq_num fields, so each row is scanned into a Std plus those two columns,
	// which are then turned into the student ID. Embedding models.Std lets sqlx fill its fields directly.
	var rows []struct {
		models.Std
		GradYear int16 `db:"grad_year"`
		SeqNum   int16 `db:"seq_num"`
	}
	query :=
	`
		SELECT s.admission_no, s.fname, s.lname, s.grade, s.grad_year, s.seq_num
		FROM parent_students ps
		JOIN students s ON s.admission_no = ps.admission_no
		WHERE ps.parent_id = $1
		ORDER BY s.fname
	`
	if err := r.DB.SelectContext(ctx, &rows, query, parentID); err != nil {
		return nil, fmt.Errorf("list children: %w", mapPgError(err))
	}

	children := make([]models.Std, 0, len(rows))
	for _, row := range rows {
		row.Std.StudentID = pkg.FormatStudentID(row.GradYear, row.SeqNum)
		children = append(children, row.Std)
	}
	return children, nil
}
