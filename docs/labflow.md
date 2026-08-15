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
