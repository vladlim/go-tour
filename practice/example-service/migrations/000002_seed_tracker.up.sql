BEGIN;

INSERT INTO users (name, email) VALUES
    ('Анна', 'anna@example.test'),
    ('Борис', 'boris@example.test'),
    ('Вера', 'vera@example.test'),
    ('Глеб', 'gleb@example.test');

INSERT INTO projects (owner_id, name, description)
SELECT id, 'Учебный сервис', 'HTTP API и PostgreSQL'
FROM users WHERE email = 'anna@example.test';
INSERT INTO projects (owner_id, name, description)
SELECT id, 'Личный сайт', 'Портфолио и блог'
FROM users WHERE email = 'boris@example.test';
INSERT INTO projects (owner_id, name, description)
SELECT id, 'Идеи', 'Пока без задач'
FROM users WHERE email = 'anna@example.test';

INSERT INTO tasks (project_id, assignee_id, title, status, priority, due_at)
SELECT p.id, u.id, seed.title, seed.status, seed.priority, seed.due_at
FROM (VALUES
    ('Учебный сервис', 'anna@example.test', 'Поднять PostgreSQL', 'done', 3, '2026-10-25 12:00:00+00'::timestamptz),
    ('Учебный сервис', 'boris@example.test', 'Сделать список задач', 'in_progress', 3, '2026-11-10 12:00:00+00'::timestamptz),
    ('Учебный сервис', 'vera@example.test', 'Добавить фильтр по статусу', 'todo', 2, NULL::timestamptz),
    ('Учебный сервис', NULL, 'Добавить пагинацию', 'todo', 1, NULL::timestamptz),
    ('Личный сайт', 'boris@example.test', 'Сверстать главную', 'done', 2, NULL::timestamptz),
    ('Личный сайт', 'vera@example.test', 'Добавить страницу контактов', 'in_progress', 2, '2026-11-15 12:00:00+00'::timestamptz),
    ('Личный сайт', NULL, 'Написать первый пост', 'todo', 1, NULL::timestamptz)
) AS seed(project_name, email, title, status, priority, due_at)
JOIN projects p ON p.name = seed.project_name
LEFT JOIN users u ON u.email = seed.email;

COMMIT;
