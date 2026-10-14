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

func (r *Repository) GetAllStudents(ctx context.Context) (*[]models.Std, error) {

	var std []models.Std
	query :=
	`
		SELECT student_id, fname, lname, mname, grade
		FROM students
	`
	if err := r.DB.GetContext(ctx, &std, query); err != nil {
		return nil, fmt.Errorf("get student: %w", mapPgError(err))
	}
	return &std, nil
}

