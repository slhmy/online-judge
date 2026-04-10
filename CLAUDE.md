# Online Judge Project

## Development Workflow

### Git Hooks

Pre-commit and pre-push hooks are configured to ensure code quality:

- **Pre-commit**: Runs `make check` (linting + type checking)
- **Pre-push**: Runs `make test` (all unit tests)

To bypass hooks temporarily (not recommended):
```bash
git commit --no-verify
git push --no-verify
```

### Make Targets

| Target | Description |
|--------|-------------|
| `make check` | Run linting and type checks (pre-commit) |
| `make test` | Run all unit tests (pre-push) |
| `make lint` | Run all linters |
| `make lint-fix` | Auto-fix lint issues |
| `make build` | Build all Go services |

## Project Structure

- `backend/` - Go center gRPC service (problem, submission, contest, notification, user, judge - all unified)
- `bff/` - Backend-for-Frontend layer
- `judge/` - Judging daemon and workers
- `frontend/` - Next.js frontend