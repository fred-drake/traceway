CREATE TABLE IF NOT EXISTS profiling_samples
(
    `project_id` UUID,
    `profile_id` UUID,
    `service_name` LowCardinality(String) DEFAULT '',
    `type` LowCardinality(String) DEFAULT '',
    `start_time` DateTime64(3),
    `end_time` DateTime64(3),
    `stack_hash` UInt64,
    `value` Int64 CODEC(ZSTD(1)),
    `labels` Map(LowCardinality(String), String),
    `server_name` LowCardinality(String) DEFAULT '',
    `app_version` LowCardinality(String) DEFAULT '',
    `trace_id` String DEFAULT '',
    `span_id` String DEFAULT '',
    INDEX idx_stack_hash stack_hash TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(start_time)
ORDER BY (project_id, type, service_name, start_time)
SETTINGS index_granularity = 8192
