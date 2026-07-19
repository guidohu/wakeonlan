package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"wakeonlan/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

var (
	loginAttemptsMu sync.Mutex
	loginAttempts   = make(map[string]int)
	loginLastSeen   = make(map[string]time.Time)
	maxAttempts     = 5
	blockDuration   = 15 * time.Minute
)

func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, cidr := range config.TrustedProxies {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func getClientIP(r *http.Request) string {
	remoteIPStr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIPStr = r.RemoteAddr
	}
	remoteIP := net.ParseIP(remoteIPStr)

	if !isTrustedProxy(remoteIP) {
		return strings.TrimSpace(remoteIPStr)
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ipStr := strings.TrimSpace(ips[i])
			ip := net.ParseIP(ipStr)
			if ip != nil && !isTrustedProxy(ip) {
				return ipStr
			}
		}
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		ipStr := strings.TrimSpace(xri)
		ip := net.ParseIP(ipStr)
		if ip != nil && !isTrustedProxy(ip) {
			return ipStr
		}
	}

	return strings.TrimSpace(remoteIPStr)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getClientIP(r)

	loginAttemptsMu.Lock()
	if lastSeen, exists := loginLastSeen[ip]; exists {
		if time.Since(lastSeen) > blockDuration {
			delete(loginAttempts, ip)
			delete(loginLastSeen, ip)
		} else if loginAttempts[ip] >= maxAttempts {
			loginAttemptsMu.Unlock()
			http.Error(w, "Too many failed login attempts", http.StatusTooManyRequests)
			return
		}
	}
	loginAttemptsMu.Unlock()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userMatch := subtle.ConstantTimeCompare([]byte(req.Username), []byte(config.AdminUser))
	passMatch := subtle.ConstantTimeCompare([]byte(req.Password), []byte(config.AdminPassword))

	if userMatch == 0 || passMatch == 0 {
		loginAttemptsMu.Lock()
		loginAttempts[ip]++
		loginLastSeen[ip] = time.Now()
		loginAttemptsMu.Unlock()

		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	loginAttemptsMu.Lock()
	delete(loginAttempts, ip)
	delete(loginLastSeen, ip)
	loginAttemptsMu.Unlock()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "wakeonlan",
		Subject:   req.Username,
	})

	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: tokenString})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return config.JWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Next()
	}
}
