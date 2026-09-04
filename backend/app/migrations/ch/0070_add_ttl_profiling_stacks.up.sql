ALTER TABLE profiling_stacks MODIFY TTL toDateTime(last_seen) + INTERVAL 30 DAY
