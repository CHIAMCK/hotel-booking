package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/chiamck/hotel-booking/internal/routes"
)

func TestHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := routes.SetupRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/health", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestRoomsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := routes.SetupRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/rooms", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}
