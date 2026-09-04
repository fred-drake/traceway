CREATE TABLE IF NOT EXISTS profiles
(
    `id` UUID,
    `project_id` UUID,
    `recorded_at` DateTime64(3),
    `duration` Int64 DEFAULT 0,
    `service_name` LowCardinality(String) DEFAULT '',
    `profile_type` LowCardinality(String) DEFAULT '',
    `sample_count` UInt64 DEFAULT 0,
    `total_value` Int64 DEFAULT 0,
    `server_name` LowCardinality(String) DEFAULT '',
    `app_version` LowCardinality(String) DEFAULT '',
    `attributes` String DEFAULT '{}',
    `storage_key` String DEFAULT '',
    `trace_id` String DEFAULT '',
    `span_id` String DEFAULT '',
    `distributed_trace_id` Nullable(UUID) DEFAULT NULL,
    INDEX idx_id id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_distributed_trace_id distributed_trace_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(recorded_at)
ORDER BY (project_id, recorded_at, service_name, profile_type)
SETTINGS index_granularity = 8192
