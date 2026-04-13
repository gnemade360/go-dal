## BaseRepository Flow

### Components

- `BaseRepository[T]` holds a database provider and mapper.
- `transactionalRepository[T]` wraps `BaseRepository` with an active transaction.

### Create Operation

```mermaid
sequenceDiagram
    participant Service
    participant Repo as BaseRepository
    participant Mapper
    participant DB as Database

    Service->>Repo: Create(ctx, entity)
    Repo->>Mapper: ToRow(entity)
    Mapper-->>Repo: column map
    Repo->>Repo: build INSERT SQL
    Repo->>DB: Exec(ctx, sql, args)
    DB-->>Repo: Result
    Repo-->>Service: error?
```

### Fetch Operation

```mermaid
sequenceDiagram
    participant Service
    participant Repo as BaseRepository
    participant DB as Database
    participant Mapper

    Service->>Repo: FindByID(ctx, id)
    Repo->>Repo: build SELECT with dialect
    Repo->>DB: QueryRow(ctx, sql, id)
    DB-->>Repo: Row
    Repo->>Mapper: FromRow(row)
    Mapper-->>Repo: *T
    Repo-->>Service: entity / error
```

### Transaction Wrapper

```mermaid
sequenceDiagram
    participant Service
    participant Repo as BaseRepository
    participant DB as Database
    participant TxRepo as transactionalRepository

    Service->>Repo: Transaction(ctx, fn)
    Repo->>DB: BeginTx(ctx, nil)
    DB-->>Repo: Transaction
    Repo->>Repo: wrap into TxRepo
    Repo-->>Service: fn(ctx, TxRepo)
    Service->>TxRepo: CRUD calls
    TxRepo->>DB: Exec/Query via transaction
    alt success
        Repo->>DB: Commit
    else failure
        Repo->>DB: Rollback
    end
```

Key Points:

- Mappers abstract column mapping to keep repositories generic.
- Transactions reuse the same repository API via `transactionalRepository` to reduce boilerplate.
