// Package models
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/Jxt-Eli/template/internal/auth"
)

type User struct {
	ID           uuid.UUID `json:"id"          db:"id"`
	Name         string    `json:"name"        db:"name"`
	Role         auth.Role `json:"role"        db:"role"`
	Email        string    `json:"email"       db:"email"`
	Phone        string    `json:"phone"       db:"phone"`
	Password     string    `json:"-"           db:"password_hash"`
	CreatedAt    time.Time `json:"created_at"  db:"created_at"`
}

type Login struct {
	Name         string    `json:"name"        db:"name"`
	Email        string    `json:"email"       db:"email"`
	Phone        string    `json:"phone"       db:"phone"`
	Password     string    `json:"-"           db:"password_hash"`
}

type Role struct {
	Name         string    `json:"name"        db:"name"`
	Email        string    `json:"email"       db:"email"`
	Role         auth.Role `json:"role"        db:"role"`
}

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
	MedCon       string    `json:"med_con"       db:"med_con"`					// Relevant medical conditions if any (eg. disabilities)
	Allergies    string    `json:"allergies"    db:"allergies"`
	Photo        string    `json:"photo"        db:"photo"`
	CreatedAt    time.Time `json:"created_at"   db:"created_at"`
	AdmissionNo  uuid.UUID `json:"admission_no" db:"admission_no"`
	GradYear     int16     `json:"grad_year"    db:"grad_year"`
	SeqNum       int16     `json:"seq_num"      db:"seq_num"` 
}

type BookPacks struct {
	PackID     string              `json:"book_name"   db:"book_name"`
	Amount       int               `json:"amount"      db:"amount"`
	StockCount   string            `json:"stock_count" db:"stock_count"`
	Grade        string            `json:"grade"       db:"grade"`
	Price        decimal.Decimal   `json:"price"       db:"price"`
	AddedAt      time.Time         `json:"added_at"    db:"added_at"`
}
