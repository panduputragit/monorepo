# Auth Service — Development Guide

This guide explains the conventions used in this service: how SQLC is set up,
how requests and responses are structured, how the handler/controller pattern
works, and how to add a new action.

---

## Table of Contents

1. [Project Layout](#project-layout)
2. [SQLC Setup](#sqlc-setup)
3. [Validation (Request)](#validation-request)
4. [Response Helpers](#response-helpers)
5. [Handler Pattern](#handler-pattern)
6. [Adding a New Action — Step by Step](#adding-a-new-action--step-by-step)
7. [Token Package](#token-package)
8. [Routes](#routes)

---

## Project Layout

```
auth-service/
├── cmd/server/main.go              # entry point — wires everything together
└── internal/
    ├── config/config.go            # reads .env
    ├── db/
    │   ├── migrations/001_init.sql # DB schema
    │   ├── query/                  # raw SQL files (SQLC reads these)
    │   │   ├── admin.sql
    │   │   └── member.sql
    │   ├── sqlc.yaml               # SQLC config
    │   └── gen/                    # auto-generated Go code (DO NOT EDIT)
    │       ├── db.go
    │       ├── models.go
    │       ├── admin.sql.go
    │       └── member.sql.go
    ├── handler/                    # one file per action, gathered in Handler
    │   ├── handler.go              # Handler struct + shared helpers
    │   ├── admin_login.go
    │   ├── admin_logout.go
    │   ├── admin_sessions.go
    │   ├── member_login.go
    │   ├── member_logout.go
    │   └── member_refresh.go
    ├── http/routes.go              # registers all routes
    ├── response/auth.go            # shared response helpers
    ├── token/paseto.go             # PASETO token maker
    └── validation/auth.go          # request structs with binding tags
```

**Rule of thumb:**
- `validation/` — what comes **in** (request shape + rules)
- `response/` — what goes **out** (success/error wrappers)
- `handler/` — what happens **in between** (one file per action)
- `db/query/` — raw SQL; `db/gen/` — generated Go from that SQL

---

## SQLC Setup

SQLC reads your SQL files and generates type-safe Go code. You never write
`db.QueryRow` or `rows.Scan` manually.

### File locations

```
internal/db/
├── sqlc.yaml           ← config
├── migrations/         ← schema (tables) — SQLC reads this to understand types
│   └── 001_init.sql
└── query/              ← named queries — SQLC turns each one into a Go function
    ├── admin.sql
    └── member.sql
```

### Config — `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query/"        # where your .sql query files live
    schema: "migrations/"    # where your table definitions live
    gen:
      go:
        package: "authdb"    # Go package name for generated code
        out: "gen/"          # where to write the generated files
        emit_json_tags: true # add `json:"..."` tags to structs
        emit_pointers_for_null_types: true  # nullable columns → *Type
```

### Writing queries

Every query needs a name comment telling SQLC what to generate:

```sql
-- name: <FunctionName> :<return-type>
```

| Return type | What SQLC generates | Use when |
|---|---|---|
| `:one` | returns one row or `sql.ErrNoRows` | `SELECT ... LIMIT 1` |
| `:many` | returns `[]Row` | `SELECT` multiple rows |
| `:exec` | returns only `error` | `INSERT`, `UPDATE`, `DELETE` |

**Examples from this service:**

```sql
-- name: GetAdminByEmail :one
SELECT id, email, password_hash
FROM admins
WHERE email = $1 LIMIT 1;
```
→ Generates `func (q *Queries) GetAdminByEmail(ctx, email) (GetAdminByEmailRow, error)`

```sql
-- name: CreateAdminSession :exec
INSERT INTO admin_sessions (admin_id, token_id, expires_at)
VALUES ($1, $2, $3);
```
→ Generates `func (q *Queries) CreateAdminSession(ctx, CreateAdminSessionParams) error`

```sql
-- name: ListAdminSessions :many
SELECT id, token_id, expires_at, revoked_at, created_at
FROM admin_sessions
WHERE admin_id = $1
ORDER BY created_at DESC;
```
→ Generates `func (q *Queries) ListAdminSessions(ctx, adminID) ([]ListAdminSessionsRow, error)`

### Parameters — `$1`, `$2`, `$3`

PostgreSQL uses `$1`, `$2`, ... for parameters (not `?` like MySQL).

- One param → passed directly as a value
- Multiple params → SQLC wraps them in a `Params` struct

```sql
-- One param: passed directly
WHERE email = $1

-- Multiple params: SQLC creates a struct
INSERT INTO admin_sessions (admin_id, token_id, expires_at)
VALUES ($1, $2, $3);
-- generates: CreateAdminSessionParams{ AdminID, TokenID, ExpiresAt }
```

### Regenerate after any SQL change

```bash
cd internal/db
sqlc generate
```

Never edit files in `gen/` — they are overwritten every time you run this.

> **Gotcha:** `sqlc generate` does NOT delete stale files. If you delete or
> rename a `.sql` file, manually delete the corresponding `gen/*.sql.go` file
> first, then regenerate. Otherwise the old generated code stays and causes
> "undefined" errors at compile time.

### Generated code example

Given this query:

```sql
-- name: GetAdminByEmail :one
SELECT id, email, password_hash FROM admins WHERE email = $1 LIMIT 1;
```

SQLC generates:

```go
// in gen/admin.sql.go

type GetAdminByEmailRow struct {
    ID           uuid.UUID `json:"id"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"password_hash"`
}

func (q *Queries) GetAdminByEmail(ctx context.Context, email string) (GetAdminByEmailRow, error) {
    row := q.db.QueryRowContext(ctx, getAdminByEmail, email)
    var i GetAdminByEmailRow
    err := row.Scan(&i.ID, &i.Email, &i.PasswordHash)
    return i, err
}
```

You call it like this in your handler:

```go
admin, err := h.query.GetAdminByEmail(c.Request.Context(), req.Email)
if errors.Is(err, sql.ErrNoRows) {
    response.Unauthorized(c, "invalid credentials")
    return
}
```

---

## Validation (Request)

Request structs live in `internal/validation/`. They define what fields are
expected and what rules they must pass.

### File: `internal/validation/auth.go`

```go
package validation

type LoginRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}
```

### Struct tag breakdown

```
`json:"email" binding:"required,email"`
  │             │        │       │
  │             │        │       └─ must be a valid email format
  │             │        └───────── field cannot be empty
  │             └────────────────── Gin reads this for validation rules
  └──────────────────────────────── JSON key name in the request body
```

### Common `binding` rules

| Rule | Meaning |
|---|---|
| `required` | field must be present and non-empty |
| `email` | must be a valid email address |
| `min=6` | string must be at least 6 characters |
| `max=100` | string must be at most 100 characters |
| `len=10` | string must be exactly 10 characters |
| `numeric` | must be a number |
| `uuid` | must be a valid UUID |
| `oneof=admin member` | must be one of these values |

### How to use in a handler

```go
var req validation.LoginRequest
if err := c.ShouldBindJSON(&req); err != nil {
    response.BadRequest(c, err.Error())
    return
}
// req.Email and req.Password are now safe to use
```

`ShouldBindJSON` reads the request body, maps it to the struct, and runs all
`binding` rules. If anything fails it returns an error — you return early with
`response.BadRequest`.

---

## Response Helpers

All responses go through `internal/response/auth.go` so the shape is always
consistent.

### File: `internal/response/auth.go`

```go
type envelope struct {
    Data  any    `json:"data,omitempty"`
    Error string `json:"error,omitempty"`
}

func OK(c *gin.Context, data any)           // 200
func Created(c *gin.Context, data any)      // 201
func BadRequest(c *gin.Context, msg string) // 400
func Unauthorized(c *gin.Context, msg string) // 401
func InternalError(c *gin.Context, msg string) // 500
```

### Response shape

**Success:**
```json
{ "data": { ... } }
```

**Error:**
```json
{ "error": "invalid credentials" }
```

The `envelope` struct uses `omitempty` so only one of `data` or `error`
appears in the output — never both.

### Usage in handlers

```go
// Success — pass any value as data
response.OK(c, gin.H{
    "access_token": tokenStr,
    "expires_in":   3600,
    "role":         "admin",
})

// Success with a struct
response.OK(c, sessions)   // sessions is []ListAdminSessionsRow

// Error — pass a message string
response.BadRequest(c, err.Error())
response.Unauthorized(c, "invalid credentials")
response.InternalError(c, "failed to save session")
```

---

## Handler Pattern

### Why one file per action?

Each action (login, logout, refresh, etc.) is in its own file but all belong
to the same `Handler` struct. This means:

- Easy to find a specific action (file name = action name)
- All actions share the same dependencies (`h.query`, `h.token`)
- Adding a new action = adding one new file, no changes elsewhere in `handler/`

### `handler.go` — the struct (controller)

```go
// internal/handler/handler.go

type Handler struct {
    query *authdb.Queries  // all DB queries
    token *token.Maker     // PASETO token maker
}

func New(db *sql.DB, tokenMaker *token.Maker) *Handler {
    return &Handler{
        query: authdb.New(db),  // wraps *sql.DB with generated query methods
        token: tokenMaker,
    }
}
```

`Handler` is the controller. Every action is a method on it. Dependencies are
injected once in `New()` and shared across all actions.

### Action file anatomy

Every action file follows the same shape:

```go
// internal/handler/admin_login.go
package handler

import (...)

// Method on *Handler — this is how it shares h.query and h.token
func (h *Handler) AdminLogin(c *gin.Context) {

    // 1. Parse + validate request
    var req validation.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return   // ← always return after a response
    }

    // 2. Query DB using SQLC-generated method
    admin, err := h.query.GetAdminByEmail(c.Request.Context(), req.Email)
    if errors.Is(err, sql.ErrNoRows) {
        response.Unauthorized(c, "invalid credentials")
        return
    }
    if err != nil {
        response.InternalError(c, "failed to fetch admin")
        return
    }

    // 3. Business logic
    if err := bcrypt.CompareHashAndPassword(
        []byte(admin.PasswordHash), []byte(req.Password),
    ); err != nil {
        response.Unauthorized(c, "invalid credentials")
        return
    }

    // 4. Side effects (create token, write to DB)
    tokenStr, payload, err := h.token.CreateToken(...)
    if err != nil { ... }

    if err := h.query.CreateAdminSession(c.Request.Context(), authdb.CreateAdminSessionParams{
        AdminID:   admin.ID,
        TokenID:   payload.ID,
        ExpiresAt: payload.ExpiredAt,
    }); err != nil {
        response.InternalError(c, "failed to save session")
        return
    }

    // 5. Return response
    response.OK(c, gin.H{
        "access_token": tokenStr,
        "expires_in":   int64(duration.Seconds()),
        "role":         "admin",
    })
}
```

### Pattern rules

1. **Always `return` after calling a response helper.** Gin does not stop
   execution automatically after `c.JSON(...)`.
2. **Check `sql.ErrNoRows` separately** from generic errors — they mean
   different things (not found vs server error).
3. **Pass `c.Request.Context()`** to all DB calls so timeouts propagate.
4. **Never return raw DB errors** to the client — they may leak schema details.

### Shared helpers inside handler.go

Helper functions used by multiple actions live in `handler.go`:

```go
// Extracts "Bearer <token>" → "<token>"
func bearerToken(header string) string { ... }

// Validates Bearer token + checks role == "admin"
// Used by AdminLogout, ListAdminSessions, ForceRevokeAdminSession
func (h *Handler) requireAdminToken(c *gin.Context) (*token.Payload, bool) {
    tokenStr := bearerToken(c.GetHeader("Authorization"))
    if tokenStr == "" {
        response.Unauthorized(c, "missing token")
        return nil, false
    }
    payload, err := h.token.VerifyToken(tokenStr)
    if err != nil {
        response.Unauthorized(c, err.Error())
        return nil, false
    }
    if payload.Role != "admin" {
        response.Unauthorized(c, "forbidden")
        return nil, false
    }
    return payload, true
}
```

Usage in any admin action:

```go
func (h *Handler) AdminLogout(c *gin.Context) {
    payload, ok := h.requireAdminToken(c)
    if !ok {
        return  // response already written inside requireAdminToken
    }
    // payload.UserID, payload.ID, etc. are now available
}
```

---

## Adding a New Action — Step by Step

Example: add `POST /admin/change-password`.

### 1. Add request struct to `validation/auth.go`

```go
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" binding:"required,min=6"`
    NewPassword     string `json:"new_password"     binding:"required,min=6"`
}
```

### 2. Write the SQL query in `query/admin.sql`

```sql
-- name: UpdateAdminPassword :exec
UPDATE admins SET password_hash = $2 WHERE id = $1;
```

### 3. Regenerate SQLC

```bash
cd internal/db && sqlc generate
```

This creates `UpdateAdminPassword(ctx, UpdateAdminPasswordParams) error` in
`gen/admin.sql.go`.

### 4. Create the action file `handler/admin_change_password.go`

```go
package handler

import (
    "database/sql"
    "errors"

    authdb "github.com/panduputragit/gym/backend/app/auth-service/internal/db/gen"
    "github.com/panduputragit/gym/backend/app/auth-service/internal/response"
    "github.com/panduputragit/gym/backend/app/auth-service/internal/validation"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
)

func (h *Handler) AdminChangePassword(c *gin.Context) {
    payload, ok := h.requireAdminToken(c)
    if !ok {
        return
    }

    var req validation.ChangePasswordRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    adminID, _ := uuid.Parse(payload.UserID)

    admin, err := h.query.GetAdminByEmail(c.Request.Context(), payload.Email)
    if errors.Is(err, sql.ErrNoRows) {
        response.Unauthorized(c, "admin not found")
        return
    }
    if err != nil {
        response.InternalError(c, "failed to fetch admin")
        return
    }

    if err := bcrypt.CompareHashAndPassword(
        []byte(admin.PasswordHash), []byte(req.CurrentPassword),
    ); err != nil {
        response.Unauthorized(c, "wrong current password")
        return
    }

    hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        response.InternalError(c, "failed to hash password")
        return
    }

    if err := h.query.UpdateAdminPassword(c.Request.Context(), authdb.UpdateAdminPasswordParams{
        ID:           adminID,
        PasswordHash: string(hashed),
    }); err != nil {
        response.InternalError(c, "failed to update password")
        return
    }

    response.OK(c, gin.H{"message": "password updated"})
}
```

### 5. Register the route in `http/routes.go`

```go
admin := router.Group("/admin")
admin.POST("/login", h.AdminLogin)
admin.POST("/logout", h.AdminLogout)
admin.POST("/change-password", h.AdminChangePassword)  // ← add here
admin.GET("/sessions", h.ListAdminSessions)
admin.DELETE("/sessions/:id", h.ForceRevokeAdminSession)
```

That's it. No changes to `handler.go`, `main.go`, or any other file.

---

## Token Package

`internal/token/paseto.go` wraps PASETO v2 (asymmetric).

### Payload

```go
type Payload struct {
    ID        string    // unique token ID — used as the key in DB tables
    UserID    string    // admin or member UUID
    Email     string
    Role      string    // "admin" | "member" | "member_refresh"
    IssuedAt  time.Time
    ExpiredAt time.Time
}
```

### Creating a token

```go
tokenStr, payload, err := h.token.CreateToken(
    userID,    // string — UUID of the user
    email,     // string
    "admin",   // role
    8*time.Hour,
)
```

Returns:
- `tokenStr` — the encoded token string to send to the client
- `payload` — the decoded payload (use `payload.ID` to store in DB,
  `payload.ExpiredAt` to set `expires_at`)

### Verifying a token

```go
payload, err := h.token.VerifyToken(tokenStr)
if err != nil {
    // token.ErrExpiredToken or token.ErrInvalidToken
}
// payload.UserID, payload.Role, payload.ID available
```

### Role convention

| Role | Token type | Where stored |
|---|---|---|
| `"admin"` | admin access token | `admin_sessions.token_id` |
| `"member"` | member access token | nowhere (stateless) |
| `"member_refresh"` | member refresh token | `member_refresh_tokens.token_id` |

---

## Routes

`internal/http/routes.go` maps HTTP methods + paths to handler methods.

```go
func RegisterRoutes(router *gin.Engine, h *handler.Handler) {
    admin := router.Group("/admin")
    admin.POST("/login",          h.AdminLogin)
    admin.POST("/logout",         h.AdminLogout)
    admin.GET("/sessions",        h.ListAdminSessions)
    admin.DELETE("/sessions/:id", h.ForceRevokeAdminSession)

    member := router.Group("/member")
    member.POST("/login",   h.MemberLogin)
    member.POST("/logout",  h.MemberLogout)
    member.POST("/refresh", h.MemberRefresh)
}
```

Route params (`:id`) are read in the handler with `c.Param("id")`:

```go
sessionID, err := uuid.Parse(c.Param("id"))
```

Query params (`?since=...`) are read with `c.Query("since")`.

---

## Quick Reference

| Task | Where |
|---|---|
| Add a new request field | `internal/validation/auth.go` |
| Add a new SQL query | `internal/db/query/*.sql` then `sqlc generate` |
| Add a new action | new file in `internal/handler/`, then register in `routes.go` |
| Change response shape | `internal/response/auth.go` |
| Change token payload | `internal/token/paseto.go` |
| Add a new route | `internal/http/routes.go` |
