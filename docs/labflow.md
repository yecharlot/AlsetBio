# LabFlow

LabFlow cubre el trabajo diario con muestras: entrada, estados, custodia y comprobación externa.

## Flujo habitual

1. Crear muestra (`POST /api/labflow/samples` o la UI)
2. Asignar y procesar (`ASSIGNED` → `IN_PROGRESS`)
3. Control de calidad (`QC_REVIEW`)
4. Liberar o marcar incidencia (`RELEASED` / `FLAGGED`)
5. Archivar cuando corresponda
6. Compartir verificación (`/verify/<id>` o QR)

## Estados

```text
RECEIVED → ASSIGNED → IN_PROGRESS → QC_REVIEW → RELEASED → ARCHIVED
                         ↘ FLAGGED ↙
```

Las flechas concretas dependen del `workflow_id` de la muestra.

## Workflows incluidos

- **default** — ciclo genérico
- **water-testing** — permite volver de QC a proceso (reensayo)
- **clinical** — si queda FLAGGED, solo puede archivarse

Listado: `GET /api/labflow/workflows`

## Custodia

Cada cambio relevante genera un evento (no se borran; una corrección es un evento nuevo). Los eventos y el snapshot de la muestra viven como bloques con CID.

## Roles

- **LAB_ADMIN** / **LAB_MANAGER** — gestión amplia
- **TECHNICIAN** — crear y avanzar operación básica
- **REVIEWER** — QC y liberación
- **CLIENT** — vista restringida a sus muestras

Token: `POST /api/labflow/auth/token`.  
Modo estricto: `LABFLOW_REQUIRE_AUTH=true`.

## UI

`/w/labflow.app.ans` — altas, listado, transiciones, contadores, timeline y QR.

## Límites conscientes

No incluye diagnóstico clínico, descubrimiento de fármacos ni acoplamiento profundo a instrumentos. Eso queda para fases posteriores del producto.
