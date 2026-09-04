CREATE TABLE IF NOT EXISTS profiling_stacks
(
    `project_id` UUID,
    `service_name` LowCardinality(String) DEFAULT '',
    `stack_hash` UInt64,
    `stack` Array(LowCardinality(String)),
    `last_seen` DateTime64(3)
)
ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (project_id, service_name, stack_hash)
SETTINGS index_granularity = 8192
