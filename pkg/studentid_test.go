package pkg

import (
	"errors"
	"testing"
)

func TestParseStudentID(t *testing.T) {
	gy, seq, err := ParseStudentID("30-0042")
	if err != nil || gy != 30 || seq != 42 {
		t.Fatalf("ParseStudentID(\"30-0042\") = %d, %d, %v; want 30, 42, nil", gy, seq, err)
	}
	if got := FormatStudentID(gy, seq); got != "30-0042" {
		t.Errorf("round trip gave %q", got)
	}

	for _, bad := range []string{"", "30", "300042", "30-42", "30-0042abc", "ab-cdef", "30-0000", "-1-0042"} {
		if _, _, err := ParseStudentID(bad); !errors.Is(err, ErrInvalidStudentID) {
			t.Errorf("ParseStudentID(%q) error = %v, want ErrInvalidStudentID", bad, err)
		}
	}
}
