package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"wakeonlan/config"
	"wakeonlan/handlers"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestHandleLogin_Success(t *testing.T) {
	config.AdminUser = "testadmin"
	config.AdminPassword = "testpassword"
	config.JWTSecret = []byte("testsecret")

	reqBody := handlers.LoginRequest{
		Username: "testadmin",
		Password: "testpassword",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleLogin)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status OK, got %v", status)
	}

	var resp handlers.AuthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Token == "" {
		t.Errorf("expected token in response, got empty")
	}
}

func TestHandleLogin_Failure(t *testing.T) {
	config.AdminUser = "testadmin"
	config.AdminPassword = "testpassword"

	reqBody := handlers.LoginRequest{
		Username: "testadmin",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleLogin)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("expected status Unauthorized, got %v", status)
	}
}

func TestHandleLogin_RateLimit(t *testing.T) {
	config.AdminUser = "testadmin"
	config.AdminPassword = "testpassword"

	handler := http.HandlerFunc(handlers.HandleLogin)

	// Make 5 failed requests
	for i := 0; i < 5; i++ {
		reqBody := handlers.LoginRequest{
			Username: "testadmin",
			Password: "wrongpassword",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12345"

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status Unauthorized on attempt %d, got %v", i+1, status)
		}
	}

	// 6th attempt should be blocked
	reqBody := handlers.LoginRequest{
		Username: "testadmin",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("expected status TooManyRequests on 6th attempt, got %v", status)
	}

	// Successful request from a DIFFERENT IP should still work
	reqBodySuccess := handlers.LoginRequest{
		Username: "testadmin",
		Password: "testpassword",
	}
	bodySuccess, _ := json.Marshal(reqBodySuccess)

	reqSuccess, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(bodySuccess))
	reqSuccess.Header.Set("Content-Type", "application/json")
	reqSuccess.RemoteAddr = "192.168.1.101:12345"

	rrSuccess := httptest.NewRecorder()
	handler.ServeHTTP(rrSuccess, reqSuccess)

	if status := rrSuccess.Code; status != http.StatusOK {
		t.Errorf("expected status OK for different IP, got %v", status)
	}
}

func TestAuthMiddleware(t *testing.T) {
	config.JWTSecret = []byte("testsecret")
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(handlers.AuthMiddleware())
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Test missing token
	req, _ := http.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected Unauthorized for missing token, got %v", rr.Code)
	}

	// Test invalid token
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected Unauthorized for invalid token, got %v", rr.Code)
	}

	// Test valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})
	tokenString, _ := token.SignedString(config.JWTSecret)

	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected OK for valid token, got %v", rr.Code)
	}
}
