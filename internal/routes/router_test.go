package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/idempotency"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/routes"
	"github.com/gin-gonic/gin"
)

type stubRoomRepository struct{}

func (stubRoomRepository) Exists(id int) (bool, error) {
	return id >= 1 && id <= 10, nil
}

type stubRoomCategoryRepository struct{}

func (stubRoomCategoryRepository) Search(params repository.RoomCategorySearchParams) (models.RoomCategorySearchPage, error) {
	return models.RoomCategorySearchPage{
		Categories: []models.RoomCategorySearchResult{
			{ID: 1, Name: "Deluxe Room", MaxPerson: 2, BasePrice: 150, AvailableCount: 2},
			{ID: 2, Name: "Executive Room", MaxPerson: 3, BasePrice: 200, AvailableCount: 1},
		},
		Pagination: models.Pagination{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      2,
			TotalPages: 1,
		},
	}, nil
}

type stubBookingRepository struct{}

func (stubBookingRepository) Create(params repository.CreateBookingParams) (models.Booking, error) {
	return models.Booking{
		ID:            99,
		RoomID:        params.RoomID,
		CustomerID:    params.CustomerID,
		StartTime:     params.CheckIn,
		EndTime:       params.CheckOut,
		Status:        "confirmed",
		TotalAmount:   750,
		PricePerNight: 150,
	}, nil
}

func (stubBookingRepository) ListBlockingByRoomOverlap(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error) {
	return nil, nil
}

type stubLocker struct{}

func (stubLocker) TryLock(ctx context.Context, key string, exp time.Duration) (func(), bool, error) {
	return func() {}, true, nil
}

type memoryIdempotencyStore struct {
	mu sync.Mutex
	m  map[string]struct{}
}

func newMemoryIdempotencyStore() *memoryIdempotencyStore {
	return &memoryIdempotencyStore{m: make(map[string]struct{})}
}

func (s *memoryIdempotencyStore) CheckIdempotent(ctx context.Context, idempotencyKey string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[idempotencyKey]
	return ok, nil
}

func (s *memoryIdempotencyStore) SetIdempotent(ctx context.Context, idempotencyKey string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[idempotencyKey] = struct{}{}
	return nil
}

var _ idempotency.BookingStore = (*memoryIdempotencyStore)(nil)

func setupTestRouter() *gin.Engine {
	return routes.SetupRouter(routes.Dependencies{
		RoomRepo:           stubRoomRepository{},
		RoomCategoryRepo:   stubRoomCategoryRepository{},
		BookingRepo:        stubBookingRepository{},
		Lock:               stubLocker{},
		BookingIdempotency: newMemoryIdempotencyStore(),
	})
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

func TestRoomAvailabilityRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/rooms/2/availability?from=2026-07-01&to=2026-07-10", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	if cc := response.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", cc)
	}
}

func TestRoomAvailabilityRouteNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/rooms/99/availability", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
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
		Pagination struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
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

	if payload.Pagination.Page != 1 || payload.Pagination.Limit != 10 || payload.Pagination.Total != 2 {
		t.Fatalf("unexpected pagination: %+v", payload.Pagination)
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

func TestRoomCategorySearchRouteInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(
		http.MethodGet,
		"/api/v1/room-categories?hotel_id=1&check_in=2026-06-10&check_out=2026-06-15&guests=2&page=0",
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

func TestCreateBookingRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := bytes.NewBufferString(`{
		"room_id": 2,
		"customer_id": 1,
		"check_in": "2026-07-01",
		"check_out": "2026-07-06"
	}`)

	router := setupTestRouter()
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/bookings", body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
}

var _ lock.DistributedLock = (*stubLocker)(nil)
