# API contracts

The initial API is versioned under `/api/v1`.

Conventions:
- JSON request and response bodies.
- UTC timestamps in RFC 3339 format.
- UUID resource identifiers.
- Consistent error envelope: `{"error":{"code":"...","message":"..."}}`.
- List endpoints use explicit pagination parameters.

OpenAPI specification will be added in M1 before monitor CRUD is considered complete.
