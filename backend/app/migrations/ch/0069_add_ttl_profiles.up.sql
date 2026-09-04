ALTER TABLE profiles MODIFY TTL toDateTime(recorded_at) + INTERVAL 30 DAY
