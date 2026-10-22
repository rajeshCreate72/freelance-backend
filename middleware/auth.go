package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for keys used to store values
// in an HTTP request's context. A private type avoids key collisions.
type contextKey string

const (
	// These keys identify the authenticated user's data in the request context.
	accountIDKey contextKey = "accountID"
	roleKey      contextKey = "role"
)

// Claims describes the data this application puts in a JWT.
//
// The login handler sets:
//   - Subject: the account ID
//   - Role: "client" or "freelancer"
//   - ExpiresAt: when the token expires
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Authenticate creates middleware that checks a request's JWT.
//
// It receives the JWT secret and returns a function that can wrap
// an HTTP handler. The wrapped handler runs only if the token is valid.
func Authenticate(jwtSecret []byte) func(http.Handler) http.Handler {
	// This function receives the next handler in the request chain,
	// such as the GET /clients/me endpoint.
	return func(next http.Handler) http.Handler {
		// This handler runs for each incoming request to the protected route.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Without the secret, the server cannot verify JWT signatures.
			if len(jwtSecret) == 0 {
				http.Error(
					w,
					"Server authentication is not configured",
					http.StatusInternalServerError,
				)
				return
			}

			// The client should send:
			// Authorization: Bearer <the-token>
			authorizationHeader := r.Header.Get("Authorization")

			// Reject the request if it does not use the Bearer format.
			if !strings.HasPrefix(authorizationHeader, "Bearer ") {
				http.Error(
					w,
					"Missing or invalid Authorization header",
					http.StatusUnauthorized,
				)
				return
			}

			// Remove "Bearer " and any accidental surrounding whitespace.
			tokenString := strings.TrimSpace(
				strings.TrimPrefix(authorizationHeader, "Bearer "),
			)

			// Reject an empty token.
			if tokenString == "" {
				http.Error(
					w,
					"Missing access token",
					http.StatusUnauthorized,
				)
				return
			}

			// This is where the parsed JWT claims will be stored.
			claims := &Claims{}

			// Parse the token, verify its signature, and check its expiry.
			token, err := jwt.ParseWithClaims(
				tokenString,
				claims,
				func(token *jwt.Token) (interface{}, error) {
					// Return the secret used by /login to sign the token.
					return jwtSecret, nil
				},
				// Accept only the HMAC SHA-256 signing method used by /login.
				jwt.WithValidMethods([]string{
					jwt.SigningMethodHS256.Alg(),
				}),
			)

			// Reject tokens that cannot be parsed, have a bad signature,
			// or are expired.
			if err != nil || token == nil || !token.Valid {
				http.Error(
					w,
					"Invalid or expired access token",
					http.StatusUnauthorized,
				)
				return
			}

			// The login code puts the account ID in the JWT Subject field.
			if claims.Subject == "" {
				http.Error(
					w,
					"Access token has no account ID",
					http.StatusUnauthorized,
				)
				return
			}

			// Only these two roles are valid in this application.
			if claims.Role != "client" && claims.Role != "freelancer" {
				http.Error(
					w,
					"Access token has an invalid role",
					http.StatusUnauthorized,
				)
				return
			}

			// Store the verified account ID in the request context,
			// so the endpoint can identify the logged-in account.
			requestContext := context.WithValue(
				r.Context(),
				accountIDKey,
				claims.Subject,
			)

			// Store the verified role too, for role-based access checks.
			requestContext = context.WithValue(
				requestContext,
				roleKey,
				claims.Role,
			)

			// Pass the request, now carrying the verified identity,
			// to the protected endpoint.
			next.ServeHTTP(w, r.WithContext(requestContext))
		})
	}
}

// AccountID retrieves the authenticated account ID from a request.
// The second return value is false if the ID was not present.
func AccountID(r *http.Request) (string, bool) {
	accountID, ok := r.Context().Value(accountIDKey).(string)
	return accountID, ok
}

// Role retrieves the authenticated account role from a request.
// The second return value is false if the role was not present.
func Role(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(roleKey).(string)
	return role, ok
}