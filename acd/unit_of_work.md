## Unit of Work Flow

### Registration

```mermaid
sequenceDiagram
    participant App
    participant UoW as UnitOfWork
    participant Factory

    App->>UoW: RegisterRepository("users", factory)
    UoW->>Factory: store callback
```

### InTransaction Helper

```mermaid
sequenceDiagram
    participant Service
    participant UoW as TransactionalUnitOfWork
    participant Tx as UnitOfWorkTransaction
    participant Repo

    Service->>UoW: InTransaction(ctx, fn)
    UoW->>UoW: Begin(ctx)
    UoW-->>Service: transactional scope
    Service->>UoW: Repository("users")
    UoW->>Repo: factory(tx)
    Repo-->>Service: transactional repo
    Service->>Repo: operations
    Service-->>UoW: fn result
    alt success
        UoW->>Tx: Commit()
    else failure
        UoW->>Tx: Rollback()
    end
```

Implementation Notes:

- Factories receive the current transaction, ensuring repositories share the same context.
- `TransactionalUnitOfWork` wraps boilerplate so service code remains concise.
