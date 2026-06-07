package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/database"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/repository"
)

func testCreateBookingParams(roomID, customerID int, checkIn, checkOut time.Time, pricePerNight float64) repository.CreateBookingParams {
	nights := int(checkOut.Sub(checkIn).Hours() / 24)
	return repository.CreateBookingParams{
		RoomID:        roomID,
		CustomerID:    customerID,
		CheckIn:       checkIn,
		CheckOut:      checkOut,
		Nights:        nights,
		PricePerNight: pricePerNight,
		TotalAmount:   pricePerNight * float64(nights),
	}
}

func openTestRedis(t *testing.T) *lock.RedisLock {
	t.Helper()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	client, err := database.ConnectRedis(redisURL)
	if err != nil {
		t.Skipf("redis unavailable: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	return lock.NewRedisLock(client)
}

func TestBookingRepositoryCreate(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)

	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	booking, err := repo.Create(testCreateBookingParams(2, 1, checkIn, checkOut, 150))
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	if booking.ID == 0 || booking.TotalAmount != 750 {
		t.Fatalf("unexpected booking: %+v", booking)
	}
}

func TestBookingRepositoryRejectsOverlap(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)

	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	if _, err := repo.Create(testCreateBookingParams(2, 1, checkIn, checkOut, 150)); err != nil {
		t.Fatalf("create first booking: %v", err)
	}

	_, err := repo.Create(testCreateBookingParams(
		2, 1,
		time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		150,
	))
	if !errors.Is(err, repository.ErrBookingOverlap) {
		t.Fatalf("expected overlap error, got %v", err)
	}
}

func TestBookingRepositoryWithRedisLock(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)

	redisLock := openTestRedis(t)
	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)

	lockKey := "booking:lock:room:3"
	acquired, err := redisLock.TryLock(context.Background(), lockKey, 10*time.Second)
	if err != nil || !acquired {
		t.Fatalf("acquire lock: acquired=%v err=%v", acquired, err)
	}
	defer redisLock.Unlock(context.Background(), lockKey)

	_, err = repo.Create(testCreateBookingParams(3, 1, checkIn, checkOut, 150))
	if err != nil {
		t.Fatalf("create booking under lock: %v", err)
	}
}

func TestBookingRepositoryList(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)

	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	if _, err := repo.Create(testCreateBookingParams(2, 1, checkIn, checkOut, 150)); err != nil {
		t.Fatalf("create booking: %v", err)
	}

	page, err := repo.List(repository.ListBookingsParams{CustomerID: 1, Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list bookings: %v", err)
	}
	if len(page.Bookings) != 2 {
		t.Fatalf("expected 2 bookings, got %d", len(page.Bookings))
	}
	if page.Pagination.Page != 1 || page.Pagination.Limit != 10 {
		t.Fatalf("unexpected pagination: %+v", page.Pagination)
	}
	if page.Bookings[0].RoomID != 2 {
		t.Fatalf("expected newest booking first, got room_id %d", page.Bookings[0].RoomID)
	}
}
