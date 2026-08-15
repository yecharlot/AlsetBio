# Desarrollo

## Tests del dominio LabFlow

```bash
go test ./internal/labflow/ -count=1
```

## Ejecutar en local

```bash
cp .env.example .env
go run ./cmd/prisma-tec
```

## Añadir una app web

1. Crear `static/apps/<nombre>/index.html`
2. Abrir `http://localhost:8080/w/<nombre>.app.ans`
3. O registrar con `POST /api/apps/register`

## Paquetes relevantes

- `internal/labflow` — muestras, custodia, authz, workflows, store CID
- `internal/node` — HTTP, bloques, integración LabFlow (`labflow_bridge.go`)
- `static/apps/labflow` — interfaz

## Criterio de cambios

Este repositorio es el vertical de laboratorio. Cambios de producto de ventas u otros negocios no van aquí; el motor compartido se documenta también en el repositorio de origen PrismaTec.
