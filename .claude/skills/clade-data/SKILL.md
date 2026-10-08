---
name: "clade-data"
description: "Database, schema, storage, logging, and data-path rules. Use when changing persistence, migrations, queries, RLS, database environments, or application logging; also before running integration tests, needing a test or dev database, running any supabase CLI command, or deciding where the dev DB lives."
---

# clade-data

逐條對照你接下來要做的事；條件成立才讀那一份，NEVER 先把整批讀進來。

- READ 若要讀或改 `supabase/migrations/**/*.sql`、`.github/workflows/**/*.yml`、`docker-compose*.yml`、`infra/**/*`、`scripts/dev-session*`、`scripts/worktree-*` 等 8 處（全表見 rules/_index.md），讀取 `rules/db-preview-env.md`

rule 內文只經本 skill 載入；清單不能代替內文。
