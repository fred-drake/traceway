CREATE TABLE IF NOT EXISTS project_user_roles (
    id SERIAL PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    role TEXT NOT NULL CHECK (role IN ('user','readonly')),
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (project_id, user_id)
)
