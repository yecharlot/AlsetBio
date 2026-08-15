# Desarrollo — AlsetBio

## Tests

```bash
go test ./...
```

## Estructura relevante

- `cmd/prisma-tec` — entrada del nodo
- `internal/node` — runtime
- `internal/labflow` — dominio LabFlow (muestras, lifecycle, custody)
- `static/apps/labflow` — UI LabFlow

## CI

GitHub Actions: download, test, build.

## Convención

No importar ni fusionar código de Alset Sales Hub en este repositorio.
