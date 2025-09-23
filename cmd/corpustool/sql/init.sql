CREATE TABLE IF NOT EXISTS namespace_name (
  id INTEGER PRIMARY KEY,
  value TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_name (
  id INTEGER PRIMARY KEY,
  value TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS tag_name (
  id INTEGER PRIMARY KEY,
  value TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repo_tag (
  id INTEGER PRIMARY KEY,
  namespace INTEGER REFERENCES namespace_name (id),
  repository INTEGER REFERENCES repository_name (id),
  tag INTEGER REFERENCES tag_name (id),
  UNIQUE (namespace, repository, tag)
);

CREATE VIEW IF NOT EXISTS refs (ref) AS
SELECT
  'quay.io/' || n.value || '/' || r.value || ':' || t.value
FROM
  repo_tag
  JOIN namespace_name AS n ON (repo_tag.namespace = n.id)
  JOIN repository_name AS r ON (repo_tag.repository = r.id)
  JOIN tag_name AS t ON (repo_tag.tag = t.id);
