# DevHub Foundation Implementation Plan

> **For agentic workers:** Execute tasks inline in this checkout. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the current API build and publish the first content module.

**Architecture:** Keep the existing Go/Gin service, add focused files under `internal/user` and `internal/content`, and maintain the OpenAPI 3 contract. PostgreSQL remains the source of truth.

**Tech Stack:** Go 1.25, Gin, GORM, PostgreSQL 18, OpenAPI 3.0.

**Spec:** `docs/superpowers/specs/2026-10-05-devhub-foundation-design.md`

## Global Constraints

- Preserve the user's three uncommitted files in `backend/internal/user`.
- Stage explicit paths only when committing.
- Every implemented endpoint is documented in `api/openapi.yaml`.

---

### Task 1: Restore registration build

**Files:** Create `backend/internal/user/register.go`, modify `backend/internal/user/errors.go`.

**Interfaces:** `RegisterRequest` binds HTTP JSON; `Service.Register(context.Context, RegisterRequest) (*Response, error)` persists a user; `ErrNotFound` is shared by repository lookups.

- [ ] Add `RegisterRequest` with length and email validation tags.
- [ ] Add `ErrNotFound` and `Service.Register` with canonical casing, bcrypt, and duplicate checks.
- [ ] Run `GOCACHE=/tmp/devhub-go-cache go test ./...` from `backend/`.
- [ ] Commit only the added and modified files.

### Task 2: Public publishing API

**Files:** Create `migrations/20261005190000_create_content_tables.sql`, `backend/internal/content/model.go`, `repository.go`, `handler.go`; modify router and main wiring, `api/openapi.yaml`.

**Interfaces:** `GET /api/v1/articles`, `GET /api/v1/articles/:slug`, `GET /api/v1/links` return the existing response envelope.

- [ ] Add migration with articles and friend links, published flags, indexes.
- [ ] Add repository query methods with pagination and published filters.
- [ ] Wire handlers and document each endpoint.
- [ ] Verify Go tests and commit explicit paths.
