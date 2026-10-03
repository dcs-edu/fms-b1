package pkg

import (
	"fmt"
	"strconv"
	"time"
)

func CalculateGradYear(grade string) ( int16, error ) {
	currentYear := time.Now().Year()
	var yearsLeft int


	switch grade {
	case "Daycare":
		yearsLeft = 12
	case "Nursery1":
		yearsLeft = 11
	case "Nursery2":
		yearsLeft = 11
	case "KG1":
		yearsLeft = 10
	case "KG2":
		yearsLeft = 10
	default:

		gradeNum, err := strconv.Atoi(grade)
			if err != nil {
				return 0, fmt.Errorf("wrong type input: %v", err)
		}
		if gradeNum >= 1 && gradeNum <= 9 {
			yearsLeft = 10 - gradeNum
		} else {
			return 0, fmt.Errorf("error! out of range")
		}
	}

	return int16((currentYear + yearsLeft) % 100), nil
}
