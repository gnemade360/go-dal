## Migration Framework Flow

### Components

- `migration.Migrator` – Orchestrates applying, rolling back, and tracking migrations.
- `migration.Provider` – Database-specific execution backend (e.g., SQLite provider).
- CLI commands in `cmd/dal-migrate` – Wrap migrator operations.

### Loading Migrations

```mermaid
sequenceDiagram
    participant CLI
    participant Loader as loadMigrations
    participant FS as Filesystem
    participant Migrator

    CLI->>Loader: loadMigrations(migrator)
    Loader->>FS: ReadDir(migrationsPath)
    FS-->>Loader: file list
    Loader->>Loader: match regex / pair up up/down
    Loader->>FS: ReadFile(up)
    Loader->>Migrator: AddMigration(Migration{Version, Up, Down})
```

### Applying Migrations (`up`)

```mermaid
sequenceDiagram
    participant CLI
    participant Migrator
    participant Provider
    participant Tx as migration.Transaction

    CLI->>Migrator: Up(ctx, target)
    Migrator->>Provider: Begin(ctx)
    Provider-->>Migrator: Tx
    Migrator->>Tx: Execute(up SQL)
    Tx->>Provider: Commit
    Migrator->>Provider: Record version
```

### Rolling Back (`down`)

```mermaid
sequenceDiagram
    participant CLI
    participant Migrator
    participant Provider
    participant Tx as migration.Transaction

    CLI->>Migrator: Down(ctx, target)
    Migrator->>Provider: Begin(ctx)
    Provider-->>Migrator: Tx
    Migrator->>Tx: Execute(down SQL)
    Tx->>Provider: Commit
    Migrator->>Provider: Update version table
```

Notes:

- Checksum validation occurs before execution when enabled.
- Providers expose `Execute` APIs used by migration steps to run batches of SQL statements.
