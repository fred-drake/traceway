ALTER TABLE profiling_samples MODIFY TTL toDateTime(start_time) + INTERVAL 30 DAY
