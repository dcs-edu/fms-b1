package handlers

import (
	"fmt"
	"net/http"

	"github.com/Jxt-Eli/template/internal/middleware"
	"github.com/Jxt-Eli/template/internal/auth"
)

func (srv *Pool) AddbookHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Fuckass logic statement that's just confusing as hell
	if claims.Role != auth.RolePrincipal && claims.Role != auth.RoleAdmin {
		http.Error(w, "forbidden: insufficient privileges", http.StatusForbidden)
		return
	}
	fmt.Fprintf(w, "Book added... addbook handler ran successfully\n")
	fmt.Fprintf(w, "Role: %v\n", claims.Role)
}
