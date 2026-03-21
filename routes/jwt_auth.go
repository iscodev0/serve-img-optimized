package routes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// verifyJWT verifica un token JWT firmado con HS256 sin depender de librerías de terceros.
func verifyJWT(tokenStr, secret string) bool {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return false
	}

	// 1. Validar la firma (Signature)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expectedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if parts[2] != expectedSignature {
		return false // Firma inválida
	}

	// 2. Validar expiración (exp) en el Payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	var payload struct {
		Exp int64 `json:"exp"`
	}
	// Ignoramos el error porque algunos JWT pueden no tener 'exp',
	// aunque para esta implementación siempre deberían tenerlo.
	json.Unmarshal(payloadJSON, &payload)

	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return false // Token expirado
	}

	return true // Token válido
}

// jwtImageAuthMiddleware protege la ruta pública exigiendo ?token=JWT
func jwtImageAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("API_TOKEN")
		if expectedToken == "" {
			// Si no hay API_TOKEN configurada, permitir paso libre (Modo Dev)
			next.ServeHTTP(w, r)
			return
		}

		// Leer el JWT de la URL (?token=...)
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "Falta token de autorización en la URL (?token=...)", http.StatusUnauthorized)
			return
		}

		if !verifyJWT(tokenStr, expectedToken) {
			http.Error(w, "Token invalido o expirado", http.StatusForbidden)
			return
		}

		// Si es válido, avanza
		next.ServeHTTP(w, r)
	}
}
