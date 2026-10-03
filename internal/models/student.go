package models

import (
	"time"

	"github.com/google/uuid"
)

type Student struct{
	StudentID    string    `json:"student_id"   db:"student_id"`
	Fname        string    `json:"fname"        db:"fname"`
	Mname        string    `json:"mname"        db:"mname"`
	Lname        string    `json:"lname"        db:"lname"`
	Grade        string    `json:"grade"        db:"grade"`
	Dob          string    `json:"dob"          db:"dob"`
	Gender       string    `json:"gender"       db:"gender"`
	Nationality  string    `json:"nationality"  db:"nationality"`
	Address      string    `json:"address"      db:"address"`	
	Guardian     string    `json:"guardian"     db:"guardian"`
	GContact     string    `json:"g_contact"    db:"g_contact"`
	Goccupation  string    `json:"g_occupation" db:"g_occupation"`
	EContact     string    `json:"e_contact"    db:"e_contact"`
	MedCon       string    `json:"med_con"      db:"med_con"`					// Relevant medical conditions if any (eg. disabilities)
	Allergies    string    `json:"allergies"    db:"allergies"`
	Photo        string    `json:"photo"        db:"photo"`
	CreatedAt    time.Time `json:"created_at"   db:"created_at"`
	AdmissionNo  uuid.UUID `json:"admission_no" db:"admission_no"`
	GradYear     int16     `json:"grad_year"    db:"grad_year"`
	SeqNum       int16     `json:"seq_num"      db:"seq_num"` 
}
