## PostgresProvider Flow

### Initialization

```mermaid
sequenceDiagram
    participant App
    participant Provider as PostgresProvider
    participant SQL as database/sql
    participant DB as PostgreSQL

    App->>Provider: New()
    App->>Provider: SetDSN(dsn)
    App->>Provider: Connect(ctx)
    Provider->>SQL: sql.Open("postgres", dsn)
    Provider->>DB: PingContext(ctx)
    Provider->>SQL: Configure pool sizes
    Provider-->>App: ready
```

### Query Execution

```mermaid
sequenceDiagram
    participant Repo
    participant Provider as PostgresProvider
    participant SQL as database/sql
    participant DB as PostgreSQL

    Repo->>Provider: Query(ctx, sql, args)
    Provider->>SQL: QueryContext(ctx, sql, args)
    SQL->>DB: Execute
    DB-->>SQL: *sql.Rows
    SQL-->>Provider: rows
    Provider-->>Repo: postgresRows wrapper
```

### Transactions

```mermaid
sequenceDiagram
    participant Repo
    participant Provider as PostgresProvider
    participant Tx as postgresTransaction
    participant SQL as database/sql

    Repo->>Provider: BeginTx(ctx, opts)
    Provider->>SQL: BeginTx(ctx, opts)
    SQL-->>Provider: *sql.Tx
    Provider-->>Repo: postgresTransaction
    Repo->>Tx: Exec/Query
    Tx->>SQL: ExecContext/QueryContext
    Repo->>Tx: Commit/Rollback
    Tx->>SQL: Commit/Rollback
```

Notes:

- The provider implements `interfaces.Database` and exposes the PostgreSQL dialect.
- `postgresTransaction` satisfies `interfaces.Transaction` by delegating to `*sql.Tx`.
