package repository_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/database"
	"github.com/chiamck/hotel-booking/internal/repository"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/hotel_booking?sslmode=disable"
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func resetSearchFixtures(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		TRUNCATE bookings, rooms, room_categories, customers, hotels RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO hotels (id, name, address) VALUES
			(1, 'Grand Plaza Hotel', '123 Main Street, Downtown');

		INSERT INTO room_categories (id, hotel_id, name, max_person, base_price) VALUES
			(1, 1, 'Deluxe Room', 2, 150.00),
			(2, 1, 'Executive Room', 3, 200.00),
			(3, 1, 'Suite', 4, 350.00);

		INSERT INTO rooms (id, hotel_id, category_id, number, status) VALUES
			(1, 1, 1, '101', 'available'),
			(2, 1, 1, '102', 'available'),
			(3, 1, 1, '105', 'available'),
			(4, 1, 1, '106', 'available'),
			(5, 1, 1, '107', 'available'),
			(6, 1, 1, '108', 'available'),
			(7, 1, 1, '109', 'available'),
			(8, 1, 1, '110', 'available'),
			(9, 1, 1, '111', 'available'),
			(10, 1, 1, '112', 'available'),
			(11, 1, 1, '113', 'available'),
			(12, 1, 2, '201', 'available'),
			(13, 1, 2, '202', 'available'),
			(14, 1, 2, '203', 'available'),
			(15, 1, 3, '301', 'available'),
			(16, 1, 3, '302', 'maintenance');

		INSERT INTO customers (id, name, email, phone) VALUES
			(1, 'Jane Doe', 'jane.doe@example.com', '+1-555-0100');

		INSERT INTO bookings (id, room_id, customer_id, start_time, end_time, status, total_amount, price_per_night) VALUES
			(1, 1, 1, '2026-06-10 00:00:00+00', '2026-06-15 00:00:00+00', 'confirmed', 750.00, 150.00);

		SELECT setval('hotels_id_seq', (SELECT MAX(id) FROM hotels));
		SELECT setval('room_categories_id_seq', (SELECT MAX(id) FROM room_categories));
		SELECT setval('rooms_id_seq', (SELECT MAX(id) FROM rooms));
		SELECT setval('customers_id_seq', (SELECT MAX(id) FROM customers));
		SELECT setval('bookings_id_seq', (SELECT MAX(id) FROM bookings));
	`); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}
}

func seedPaginationFixtures(t *testing.T, db *sql.DB) {
	t.Helper()

	resetSearchFixtures(t, db)

	if _, err := db.Exec(`
		INSERT INTO room_categories (hotel_id, name, max_person, base_price)
		SELECT 1, 'Category ' || n, 2, 100.00 + n
		FROM generate_series(4, 12) AS n;

		INSERT INTO rooms (hotel_id, category_id, number, status)
		SELECT 1, rc.id, '400-' || rc.id, 'available'
		FROM room_categories rc
		WHERE rc.name LIKE 'Category %';
	`); err != nil {
		t.Fatalf("insert pagination fixtures: %v", err)
	}
}

func TestRoomCategoryRepositorySearch(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)

	repo := repository.NewRoomCategoryRepository(db)
	checkIn := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	result, err := repo.Search(repository.RoomCategorySearchParams{
		HotelID:  1,
		Guests:   2,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Page:     1,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("search categories: %v", err)
	}

	if len(result.Categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(result.Categories))
	}

	if result.Categories[0].Name != "Deluxe Room" || result.Categories[0].AvailableCount != 10 {
		t.Fatalf("unexpected deluxe result: %+v", result.Categories[0])
	}

	if result.Categories[1].Name != "Executive Room" || result.Categories[1].AvailableCount != 3 {
		t.Fatalf("unexpected executive result: %+v", result.Categories[1])
	}

	if result.Categories[2].Name != "Suite" || result.Categories[2].AvailableCount != 1 {
		t.Fatalf("unexpected suite result: %+v", result.Categories[2])
	}
}

func TestRoomCategoryRepositorySearchPagination(t *testing.T) {
	db := openTestDB(t)
	seedPaginationFixtures(t, db)

	repo := repository.NewRoomCategoryRepository(db)
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)

	page1, err := repo.Search(repository.RoomCategorySearchParams{
		HotelID:  1,
		Guests:   2,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Page:     1,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}

	if len(page1.Categories) != 10 {
		t.Fatalf("expected 10 categories on page 1, got %d", len(page1.Categories))
	}

	page2, err := repo.Search(repository.RoomCategorySearchParams{
		HotelID:  1,
		Guests:   2,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Page:     2,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}

	if len(page2.Categories) != 2 {
		t.Fatalf("expected 2 categories on page 2, got %d", len(page2.Categories))
	}
}

func TestRoomCategoryRepositorySearchExcludesGuestMismatch(t *testing.T) {
	db := openTestDB(t)
	resetSearchFixtures(t, db)

	repo := repository.NewRoomCategoryRepository(db)
	checkIn := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	result, err := repo.Search(repository.RoomCategorySearchParams{
		HotelID:  1,
		Guests:   5,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Page:     1,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("search categories: %v", err)
	}

	if len(result.Categories) != 0 {
		t.Fatalf("expected no categories for 5 guests, got %d", len(result.Categories))
	}
}
