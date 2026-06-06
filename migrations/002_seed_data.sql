INSERT INTO hotels (id, name, address) VALUES
    (1, 'Grand Plaza Hotel', '123 Main Street, Downtown');

SELECT setval('hotels_id_seq', (SELECT MAX(id) FROM hotels));

INSERT INTO room_categories (id, hotel_id, name, max_person, base_price) VALUES
    (1, 1, 'Deluxe Room', 2, 150.00),
    (2, 1, 'Executive Room', 3, 200.00),
    (3, 1, 'Suite', 4, 350.00);

SELECT setval('room_categories_id_seq', (SELECT MAX(id) FROM room_categories));

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

SELECT setval('rooms_id_seq', (SELECT MAX(id) FROM rooms));

INSERT INTO customers (id, name, email, phone) VALUES
    (1, 'Jane Doe', 'jane.doe@example.com', '+1-555-0100');

SELECT setval('customers_id_seq', (SELECT MAX(id) FROM customers));

INSERT INTO bookings (id, room_id, customer_id, start_time, end_time, status, total_amount, price_per_night) VALUES
    (1, 1, 1, '2026-06-10 00:00:00+00', '2026-06-15 00:00:00+00', 'confirmed', 750.00, 150.00);

SELECT setval('bookings_id_seq', (SELECT MAX(id) FROM bookings));
