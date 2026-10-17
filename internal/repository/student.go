package repository

import (
	"context"
	"fmt"

	"github.com/Jxt-Eli/template/internal/models"
	"github.com/Jxt-Eli/template/pkg"
)

func (r *Repository) FullStudentProfile(ctx context.Context, studentProfile *models.Student) (*models.Student, error){
	query :=
	`
		SELECT * from students
		WHERE grad_year = $1 AND seq_num = $2
	`
	if err := r.DB.SelectContext(ctx, &studentProfile, query, studentProfile.GradYear, studentProfile.SeqNum); err != nil {
		return nil, fmt.Errorf("FullStudentProfile error: %w", mapPgError(err))
	}
	return studentProfile, nil
}
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

// ShowStudentDebts is every unfinished bill of every student. Same rules as StudentBills, so the
// owing list and a parent's bills screen always agree: only completed payments count, and no
// is_active filter (switching a utility off must not hide money still owed on it).
func (r *Repository) ShowStudentDebts(ctx context.Context) ([]models.StudentDebt, error) {
	// p.status = 'completed' must sit in the LEFT JOIN's ON. In HAVING, p.status is neither grouped
	// nor inside an aggregate, so Postgres rejects the query. In WHERE, students with no payments at all
	// (p.status is NULL) would vanish, and they are exactly the ones owing the most.
	query :=
	`
		SELECT s.admission_no, s.fname, s.mname, s.lname, s.grade, s.grad_year, s.seq_num,
			up.bill_id,
			ut.util_name,
			up.sem,
			up.amount AS price,
			COALESCE(SUM(p.amount), 0) AS paid,
			up.amount - COALESCE(SUM(p.amount), 0) AS owing
		FROM students s
		CROSS JOIN utility_prices up
		JOIN utilities ut ON ut.id = up.util_id
		LEFT JOIN payments p
		ON p.admission_no = s.admission_no
		AND p.bill_id = up.bill_id
		AND p.status = 'completed'
		GROUP BY s.admission_no, s.fname, s.mname, s.lname, s.grade, s.grad_year, s.seq_num,
		up.bill_id, ut.util_name, up.sem, up.amount
		HAVING up.amount - COALESCE(SUM(p.amount), 0) > 0
		ORDER BY s.grade, s.lname, s.admission_no, ut.util_name
	`
	// same scan as GetAllStudents: grad_year and seq_num come along so the student ID can be built
	var rows []struct {
		models.StudentDebt
		GradYear int16 `db:"grad_year"`
		SeqNum   int16 `db:"seq_num"`
	}
	if err := r.DB.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("ShowStudentDebts error: %w", mapPgError(err))
	}

	// non-nil so "nobody owes anything" encodes as [] in JSON, not null
	debts := make([]models.StudentDebt, 0, len(rows))
	for _, row := range rows {
		row.StudentDebt.StudentID = pkg.FormatStudentID(row.GradYear, row.SeqNum)
		debts = append(debts, row.StudentDebt)
	}
	return debts, nil
}
