package repository

import (
	"context"
	"fmt"
	
	"github.com/Jxt-Eli/template/pkg"
	"github.com/Jxt-Eli/template/internal/models"
)

// GetStd finds a student by the two columns their student ID is built from.
// It also loads admission_no, so callers can go on to query the tables that reference it.
func (r *Repository) GetStd(ctx context.Context, gradYear, seqNum int16) (*models.Std, error) {
	var std models.Std
	query :=
	`
		SELECT admission_no, fname, lname, grade
		FROM students
		WHERE grad_year = $1 AND seq_num = $2
	`
	if err := r.DB.GetContext(ctx, &std, query, gradYear, seqNum); err != nil {
		return nil, fmt.Errorf("get student: %w", mapPgError(err))
	}
	std.StudentID = pkg.FormatStudentID(gradYear, seqNum)
	return &std, nil
}

// GetAllStudents scans the same way as ListChildren: student_id is not a column, so grad_year and
// seq_num are read alongside each Std and turned into the ID in Go.
func (r *Repository) GetAllStudents(ctx context.Context) ([]models.Std, error) {
	var rows []struct {
		models.Std
		GradYear int16 `db:"grad_year"`
		SeqNum   int16 `db:"seq_num"`
	}
	query :=
	`
		SELECT admission_no, fname, lname, grade, grad_year, seq_num
		FROM students
		ORDER BY grade, lname, fname
	`
	if err := r.DB.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list students: %w", mapPgError(err))
	}

	students := make([]models.Std, 0, len(rows))
	for _, row := range rows {
		row.Std.StudentID = pkg.FormatStudentID(row.GradYear, row.SeqNum)
		students = append(students, row.Std)
	}
	return students, nil
}

