CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_no_overlap;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_no_overlap
    EXCLUDE USING gist (
        room_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    )
    WHERE (status IN ('pending', 'confirmed'));
