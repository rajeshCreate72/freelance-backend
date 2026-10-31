package middleware

import "net/http"

// RequireRole handles the enpoint access to user according to role(client and freelancer)
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
			role, ok := r.Context().Value(roleKey).(string)

			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			for _, allowed := range allowedRoles {
				if role == allowed {
					next. ServeHTTP(w, r)
				}
			}

			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}