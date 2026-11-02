BEGIN;

-- Убираем готовые индексы внешних ключей для самостоятельных экспериментов.
-- PRIMARY KEY и UNIQUE(email) сохраняются.
DROP INDEX projects_owner_id_idx;
DROP INDEX tasks_project_id_idx;
DROP INDEX tasks_assignee_id_idx;

INSERT INTO users (name, email, created_at)
SELECT 'Пользователь ' || n, 'user' || n || '@load.example.test',
       timestamptz '2025-01-01 00:00:00+00' + n * interval '1 minute'
FROM generate_series(1, 10000) AS s(n);

INSERT INTO projects (owner_id, name, description, created_at)
SELECT u.id, 'Проект ' || n, 'Проект для экспериментов с запросами и индексами',
       timestamptz '2025-02-01 00:00:00+00' + n * interval '1 hour'
FROM generate_series(1, 2000) AS s(n)
JOIN users u ON u.email = 'user' || (1 + (n * 7919 % 10000)) || '@load.example.test';

-- Массивы реальных ID позволяют не зависеть от состояния identity sequences.
-- Последние 20% пользователей и 10% проектов остаются без задач.
WITH user_ids AS (
    SELECT array_agg(id ORDER BY id) AS ids
    FROM users WHERE email LIKE '%@load.example.test'
), project_ids AS (
    SELECT array_agg(id ORDER BY id) AS ids
    FROM projects WHERE name LIKE 'Проект %'
), generated AS (
    SELECT n,
           ('x' || substr(md5(n::text || 'project'), 1, 8))::bit(32)::bigint AS project_hash,
           ('x' || substr(md5(n::text || 'assignee'), 1, 8))::bit(32)::bigint AS assignee_hash,
           ('x' || substr(md5(n::text || 'status'), 1, 8))::bit(32)::bigint AS status_hash
    FROM generate_series(1, 1000000) AS s(n)
)
INSERT INTO tasks (project_id, assignee_id, title, description, status, priority, due_at, created_at)
SELECT p.ids[1 + (g.project_hash % 1800)::int],
       CASE WHEN g.n % 5 = 0 THEN NULL ELSE u.ids[1 + (g.assignee_hash % 8000)::int] END,
       'Задача ' || g.n,
       repeat('Описание учебной задачи ' || g.n || '. ', 3),
       CASE WHEN g.status_hash % 100 < 80 THEN 'done'
            WHEN g.status_hash % 100 < 95 THEN 'todo'
            ELSE 'in_progress' END,
       (1 + g.assignee_hash % 3)::smallint,
       CASE WHEN g.n % 4 = 0 THEN NULL
            ELSE timestamptz '2025-01-01 00:00:00+00' + (g.n % 730) * interval '1 day' END,
       timestamptz '2025-01-01 00:00:00+00' + (g.project_hash % 63072000) * interval '1 second'
FROM generated g CROSS JOIN user_ids u CROSS JOIN project_ids p;

ANALYZE users;
ANALYZE projects;
ANALYZE tasks;

COMMIT;
