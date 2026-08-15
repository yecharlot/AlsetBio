# Alset LabFlow

LabFlow gestiona el ciclo de vida operacional de **muestras de laboratorio**.

## Alcance MVP

- Identidad de muestra (`BIO-YYYY-######`)
- Máquina de estados (RECEIVED → … → RELEASED / ARCHIVED, FLAGGED)
- Cadena de custodia (eventos append-only)
- Verificación pública controlada (sin datos sensibles)
- Dashboard operativo

## Fuera de alcance (por ahora)

IA diagnóstica, descubrimiento de fármacos, blockchain, integración profunda con instrumentos.

## Estados

```
RECEIVED → ASSIGNED → IN_PROGRESS → QC_REVIEW → RELEASED → ARCHIVED
                         ↘ FLAGGED ↙
```

Transiciones arbitrarias no permitidas.

## Integración con el Nodo Alset

- Identidad de laboratorio / operadores: agentes y tokens del nodo
- Evidencia: referencias CID / RootCID cuando aplique
- UI: app estática servida en `/w/labflow.app.ans` (en progreso)

## Disclaimer

AlsetBio / LabFlow **no es un dispositivo médico** y **no proporciona diagnóstico médico**.


## Roles (MVP)

| Rol | Puede |
|-----|-------|
| LAB_ADMIN | Todo |
| LAB_MANAGER | Todo en su ámbito |
| TECHNICIAN | Crear, ASSIGNED, IN_PROGRESS, FLAGGED |
| REVIEWER | QC_REVIEW, RELEASED, FLAGGED, ARCHIVED |
| CLIENT | Ver solo sus muestras (client_id) |

Auth: `Authorization: Bearer <token>` desde `POST /api/labflow/auth/token`.
En desarrollo, sin token: cabeceras `X-Lab-Role`, `X-Lab-Org`, `X-Lab-Actor`.
Producción: `LABFLOW_REQUIRE_AUTH=true`.


## Workflows configurables

| ID | Uso |
|----|-----|
| `default` | Ciclo genérico LabFlow |
| `water-testing` | Permite retest QC → IN_PROGRESS |
| `clinical` | FLAGGED solo puede ir a ARCHIVED |

Crear muestra con `workflow_id`. Listar: `GET /api/labflow/workflows`.
