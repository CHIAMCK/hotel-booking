package repository_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/database"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/repository"
)

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

func applyBookingConstraints(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS btree_gist;

		ALTER TABLE bookings
			ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

		DROP INDEX IF EXISTS idx_bookings_idempotency_key;

		ALTER TABLE bookings
			DROP CONSTRAINT IF EXISTS bookings_no_overlap;

		ALTER TABLE bookings
			ADD CONSTRAINT bookings_no_overlap
			EXCLUDE USING gist (
				room_id WITH =,
				tstzrange(start_time, end_time, '[)') WITH &&
			)
			WHERE (status IN ('pending', 'confirmed'));
	`); err != nil {
		t.Fatalf("apply booking constraints: %v", err)
	}
}

func TestBookingRepositoryCreate(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)
	applyBookingConstraints(t, db)

	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	booking, err := repo.Create(repository.CreateBookingParams{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
	})
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
	applyBookingConstraints(t, db)

	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	if _, err := repo.Create(repository.CreateBookingParams{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
	}); err != nil {
		t.Fatalf("create first booking: %v", err)
	}

	_, err := repo.Create(repository.CreateBookingParams{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		CheckOut:   time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected overlap error, got nil")
	}
}

func TestBookingRepositoryWithRedisLock(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)
	applyBookingConstraints(t, db)

	redisLock := openTestRedis(t)
	repo := repository.NewBookingRepository(db)
	checkIn := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)

	unlock, acquired, err := redisLock.TryLock(context.Background(), "booking:lock:room:3", 10*time.Second)
	if err != nil || !acquired {
		t.Fatalf("acquire lock: acquired=%v err=%v", acquired, err)
	}
	defer unlock()

	_, err = repo.Create(repository.CreateBookingParams{
		RoomID:     3,
		CustomerID: 1,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
	})
	if err != nil {
		t.Fatalf("create booking under lock: %v", err)
	}
}
