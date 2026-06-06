-- Idempotency replay is handled in Redis; drop DB unique constraint on idempotency_key.
DROP INDEX IF EXISTS idx_bookings_idempotency_key;
