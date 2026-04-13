## Migration CLI (`dal-migrate`)

The CLI under `cmd/dal-migrate` manages schema migrations using the DAL migration framework.

### Building

```bash
go build -o bin/dal-migrate ./cmd/dal-migrate
```

### Configuration

- Defaults are read from `.migrate.yml` (if present) and `MIGRATE_*` environment variables.
- Key flags:
  - `--driver`: database driver (`sqlite3` supported today).
  - `--dsn`: connection string.
  - `--migrations`: path to SQL migration files.
  - `--table`: migrations tracking table name.

### Common Commands

- `dal-migrate up` – Apply pending migrations (use `--to` for a target version).
- `dal-migrate down` – Roll back the latest migration (`--to` to drop to a specific version).
- `dal-migrate status` – Show applied vs. pending migrations.
- `dal-migrate create <name>` – Scaffold timestamped migration files.

### Migration Layout

The loader recognizes timestamp-based (`YYYYMMDDHHMMSS_name.up.sql`) and Flyway-style (`V1__name.sql`) scripts. Down scripts are optional when `--require-down=false`.
