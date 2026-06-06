package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/routes"
	"github.com/gin-gonic/gin"
)

type stubRoomRepository struct{}

func (stubRoomRepository) List() ([]models.Room, error) {
	return []models.Room{}, nil
}

type stubRoomCategoryRepository struct{}

func (stubRoomCategoryRepository) Search(repository.RoomCategorySearchParams) ([]models.RoomCategorySearchResult, error) {
	return []models.RoomCategorySearchResult{
		{ID: 1, Name: "Deluxe Room", MaxPerson: 2, BasePrice: 150, AvailableCount: 2},
		{ID: 2, Name: "Executive Room", MaxPerson: 3, BasePrice: 200, AvailableCount: 1},
	}, nil
}

func setupTestRouter() *gin.Engine {
	return routes.SetupRouter(
		stubRoomRepository{},
		stubRoomCategoryRepository{},
	)
}

func TestHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
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

	router := setupTestRouter()
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

func TestRoomCategorySearchRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/room-categories?hotel_id=1&check_in=2026-06-10&check_out=2026-06-15&guests=2",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var payload struct {
		Categories []struct {
			Name           string  `json:"name"`
			AvailableCount int     `json:"available_count"`
			BasePrice      float64 `json:"base_price"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(payload.Categories))
	}

	if payload.Categories[0].Name != "Deluxe Room" || payload.Categories[0].AvailableCount != 2 {
		t.Fatalf("unexpected deluxe availability: %+v", payload.Categories[0])
	}

	if payload.Categories[1].Name != "Executive Room" || payload.Categories[1].AvailableCount != 1 {
		t.Fatalf("unexpected executive availability: %+v", payload.Categories[1])
	}
}

func TestRoomCategorySearchRouteValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/room-categories?hotel_id=1&check_in=2026-06-15&check_out=2026-06-10&guests=2",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}
