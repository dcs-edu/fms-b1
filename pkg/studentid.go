package pkg

import (
	"errors"
	"fmt"
)

var ErrInvalidStudentID = errors.New("student id must look like 30-0042")

// FormatStudentID and ParseStudentID are the only two places that know the
// "YY-NNNN" layout, so the format can't drift between writing and reading it.
func FormatStudentID(gradYear, seqNum int16) string {
	return fmt.Sprintf("%02d-%04d", gradYear, seqNum)
}

// holy shit, didn't know my comments document the functions lol...
// anyways, ts just parses the stuff you give it into a student id to be stored in the db
func ParseStudentID(id string) (gradYear, seqNum int16, err error) {
	var rest string
	// %s soaks up anything after the number, so "30-0042abc" is rejected instead of silently accepted
	n, _ := fmt.Sscanf(id+"\n", "%2d-%4d%s", &gradYear, &seqNum, &rest)
	if n != 2 || len(id) != 7 || gradYear < 0 || seqNum < 1 {
		return 0, 0, fmt.Errorf("%w: got %q", ErrInvalidStudentID, id)
	}
	return gradYear, seqNum, nil
}
