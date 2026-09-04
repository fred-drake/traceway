CREATE TABLE IF NOT EXISTS project_user_roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    role TEXT NOT NULL CHECK (role IN ('user','readonly')),
    created_at DATETIME NOT NULL,
    UNIQUE (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_project_user_roles_user_id ON project_user_roles(user_id);
