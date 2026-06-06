CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_idempotency_key
    ON bookings (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_no_overlap;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_no_overlap
    EXCLUDE USING gist (
        room_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    )
    WHERE (status IN ('pending', 'confirmed'));
