# Recurso Project Instructions

These instructions are foundational mandates for the Recurso project. They take precedence over general defaults.

## Engineering Standards

We follow the "Agent Skills" framework for production-grade engineering.

### Always-On Skills (Persistent Context)

The following skills are always active and must be followed for every change:

@/Users/swapnull/Documents/Workspace/recur-so/.gemini/skills/incremental-implementation/SKILL.md
@/Users/swapnull/Documents/Workspace/recur-so/.gemini/skills/code-review-and-quality/SKILL.md

### Development Lifecycle

We use a gated workflow for all significant features and changes:

1. **`/spec`**: Write a structured specification before any code.
2. **`/planning`**: Decompose the spec into small, verifiable tasks.
3. **`/build`**: Implement tasks one at a time incrementally.
4. **`/test`**: Use TDD to ensure correctness.
5. **`/review`**: Perform a five-axis review before completion.
6. **`/ship`**: Use the pre-launch checklist.

### Project Specifics

- **Language**: Go 1.20+
- **Architecture**: Hexagonal (Internal Core vs Adapters)
- **Database**: PostgreSQL (Migrations in `migrations/`)
- **Ledger**: TigerBeetle
- **Frontend**: React (Vite) in `frontend/`

## Boundaries

- **Always**: Write tests for new logic, follow naming conventions, update `docs/` if architecture changes.
- **Ask First**: Adding new top-level directories, changing core domain models, adding major dependencies.
- **Never**: Commit secrets, bypass the type system without extreme justification.
