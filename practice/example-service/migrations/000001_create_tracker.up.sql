BEGIN;

CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL CHECK (length(trim(name)) > 0),
    email text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name text NOT NULL CHECK (length(trim(name)) > 0),
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX projects_owner_id_idx ON projects(owner_id);

CREATE TABLE tasks (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    assignee_id bigint REFERENCES users(id) ON DELETE SET NULL,
    title text NOT NULL CHECK (length(trim(title)) > 0),
    description text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'todo'
        CHECK (status IN ('todo', 'in_progress', 'done')),
    priority smallint NOT NULL DEFAULT 2 CHECK (priority BETWEEN 1 AND 3),
    due_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tasks_project_id_idx ON tasks(project_id);
CREATE INDEX tasks_assignee_id_idx ON tasks(assignee_id);

COMMIT;
