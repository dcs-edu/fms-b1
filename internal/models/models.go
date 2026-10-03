// Package models
package models
//
// import (
// 	"time"
//
// 	"github.com/google/uuid"
// 	"github.com/shopspring/decimal"
//
// 	"github.com/Jxt-Eli/template/internal/auth"
// )
//
// type User struct {
// 	ID           uuid.UUID `json:"id"          db:"id"`
// 	Name         string    `json:"name"        db:"name"`
// 	Role         auth.Role `json:"role"        db:"role"`
// 	Email        string    `json:"email"       db:"email"`
// 	Phone        string    `json:"phone"       db:"phone"`
// 	Password     string    `json:"-"           db:"password_hash"`
// 	CreatedAt    time.Time `json:"created_at"  db:"created_at"`
// }
//
// type Login struct {
// 	Name         string    `json:"name"        db:"name"`
// 	Email        string    `json:"email"       db:"email"`
// 	Phone        string    `json:"phone"       db:"phone"`
// 	Password     string    `json:"-"           db:"password_hash"`
// }
//
// type LoginResponse struct {
// 	Name         string    `json:"name"        db:"name"`
// 	Email        string    `json:"email"       db:"email"`
// 	Phone        string    `json:"phone"       db:"phone"`
// 	Password     string    `json:"-"           db:"password_hash"`
// }
//
// type Role struct {
// 	Name         string    `json:"name"        db:"name"`
// 	Email        string    `json:"email"       db:"email"`
// 	Role         auth.Role `json:"role"        db:"role"`
// }
//
// type Student struct{
// 	StudentID    string    `json:"student_id"   db:"student_id"`
// 	Fname        string    `json:"fname"        db:"fname"`
// 	Mname        string    `json:"mname"        db:"mname"`
// 	Lname        string    `json:"lname"        db:"lname"`
// 	Grade        string    `json:"grade"        db:"grade"`
// 	Dob          string    `json:"dob"          db:"dob"`
// 	Gender       string    `json:"gender"       db:"gender"`
// 	Nationality  string    `json:"nationality"  db:"nationality"`
// 	Address      string    `json:"address"      db:"address"`	
// 	Guardian     string    `json:"guardian"     db:"guardian"`
// 	GContact     string    `json:"g_contact"    db:"g_contact"`
// 	Goccupation  string    `json:"g_occupation" db:"g_occupation"`
// 	EContact     string    `json:"e_contact"    db:"e_contact"`
// 	MedCon       string    `json:"med_con"      db:"med_con"`					// Relevant medical conditions if any (eg. disabilities)
// 	Allergies    string    `json:"allergies"    db:"allergies"`
// 	Photo        string    `json:"photo"        db:"photo"`
// 	CreatedAt    time.Time `json:"created_at"   db:"created_at"`
// 	AdmissionNo  uuid.UUID `json:"admission_no" db:"admission_no"`
// 	GradYear     int16     `json:"grad_year"    db:"grad_year"`
// 	SeqNum       int16     `json:"seq_num"      db:"seq_num"` 
// }
//
// // sets of books are sold as a package for each academic year
// type BookPacks struct {
// 	PackID     string              `json:"book_name"   db:"book_name"`
// 	Amount       int               `json:"amount"      db:"amount"`
// 	StockCount   string            `json:"stock_count" db:"stock_count"`
// 	Grade        string            `json:"grade"       db:"grade"`
// 	Price        decimal.Decimal   `json:"price"       db:"price"`
// 	AddedAt      time.Time         `json:"added_at"    db:"added_at"`
// }
//
// // Utility is anything a student is billed for each semester (tuition, PTA dues, ...).
// type Utility struct {
// 	ID           int64     `json:"id"          db:"id"`
// 	UtilName     string    `json:"util_name"   db:"util_name"`
// 	IsActive     bool      `json:"is_active"   db:"is_active"`
// }
//
// // UtilityPrice is the price of one utility for one semester. Every student owes every bill of a semester.
// type UtilityPrice struct {
// 	BillID       int64             `json:"bill_id"     db:"bill_id"`
// 	UtilID       int64             `json:"util_id"     db:"util_id"`
// 	Sem          string            `json:"sem"         db:"sem"`
// 	Amount       decimal.Decimal   `json:"amount"      db:"amount"`
// }
//
// // Payment is money a student paid towards a bill. A bill can be settled over several payments.
// type Payment struct {
// 	TxnID        uuid.UUID         `json:"txn_id"       db:"txn_id"`
// 	BillID       int64             `json:"bill_id"      db:"bill_id"`
// 	AdmissionNo  uuid.UUID         `json:"admission_no" db:"admission_no"`
// 	Amount       decimal.Decimal   `json:"amount"       db:"amount"`
// 	PaidAt       time.Time         `json:"paid_at"      db:"paid_at"`
// 	Reason       *string           `json:"reason"       db:"reason"`		// nullable: nil <-> NULL
// }
