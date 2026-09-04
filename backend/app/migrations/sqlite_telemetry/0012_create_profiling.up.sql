CREATE TABLE IF NOT EXISTS profiling_stacks (
    project_id TEXT NOT NULL,
    service_name TEXT NOT NULL DEFAULT '',
    stack_hash INTEGER NOT NULL,
    stack TEXT NOT NULL DEFAULT '[]',
    last_seen DATETIME NOT NULL,
    UNIQUE(project_id, service_name, stack_hash)
);
CREATE TABLE IF NOT EXISTS profiling_samples (
    project_id TEXT NOT NULL,
    profile_id TEXT NOT NULL,
    service_name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    stack_hash INTEGER NOT NULL,
    value INTEGER NOT NULL DEFAULT 0,
    labels TEXT NOT NULL DEFAULT '{}',
    server_name TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    trace_id TEXT NOT NULL DEFAULT '',
    span_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_profiling_samples_query ON profiling_samples(project_id, type, service_name, start_time);
CREATE TABLE IF NOT EXISTS profiles (
    id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    recorded_at DATETIME NOT NULL,
    duration INTEGER NOT NULL DEFAULT 0,
    service_name TEXT NOT NULL DEFAULT '',
    profile_type TEXT NOT NULL DEFAULT '',
    sample_count INTEGER NOT NULL DEFAULT 0,
    total_value INTEGER NOT NULL DEFAULT 0,
    server_name TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    attributes TEXT NOT NULL DEFAULT '{}',
    storage_key TEXT NOT NULL DEFAULT '',
    trace_id TEXT NOT NULL DEFAULT '',
    span_id TEXT NOT NULL DEFAULT '',
    distributed_trace_id TEXT DEFAULT NULL
);
CREATE INDEX IF NOT EXISTS idx_profiles_project_recorded ON profiles(project_id, recorded_at);
