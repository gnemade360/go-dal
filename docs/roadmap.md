## Roadmap & Project Status

The core abstractions (interfaces, PostgreSQL provider, repository foundation, migration engine, example app) are in place. The library is usable for projects targeting PostgreSQL today, but several enhancements remain outstanding.

### Completed

- PostgreSQL provider with connection pooling and dialect support
- Generic repository with specifications and transaction helpers
- Migration framework and CLI with checksum validation
- Basic CRUD example service

### In Progress / Planned

- Additional providers (MySQL, MongoDB, SAP HANA)
- Redis-backed caching layer
- Query builder utilities
- Soft deletes and audit logging
- Connection pooling tuning and metrics

Contributions are welcome. See the root `README.md` for contribution guidelines and existing backlog items.
