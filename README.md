# AlsetBio

**Nodo Alset especializado para laboratorios y biotecnología.**

AlsetBio es un vertical científico del ecosistema Alset. Parte del mismo runtime de nodo (identidad, agentes, RootCID/CID, API, persistencia, P2P) y lo orienta a:

- laboratorios
- gestión de muestras
- trazabilidad y cadena de custodia
- workflows científicos
- evidencia verificable

> **Independiente de PrismaTec / Alset Sales Hub.**  
> Este repositorio no forma parte del producto comercial de ventas. PrismaTec permanece como despliegue de Sales Hub.

## Qué es LabFlow

**Alset LabFlow** es la primera aplicación de AlsetBio: ciclo de vida operacional de muestras de laboratorio.

MVP (en construcción / base lista en este repo):

- Sample intake e identidad (`BIO-YYYY-######`)
- Estados: RECEIVED → ASSIGNED → IN_PROGRESS → QC_REVIEW → RELEASED → ARCHIVED (+ FLAGGED)
- Chain of custody (eventos append-only)
- Verificación pública controlada
- Dashboard de laboratorio

**No es un dispositivo médico.** No realiza diagnóstico clínico ni prescribe tratamientos. Es una plataforma de workflow y procedencia de laboratorio.

## Arquitectura

```
ALSET NODE (runtime en este repo)
  identity | agents | RootCID | CID | auth | API | events | persistence | P2P
       |
   AlsetBio vertical
       |
   LabFlow  (→ luego BioPassport, BioDecision, BioResearch)
```

El código del nodo vive bajo `cmd/` e `internal/` (mismo diseño modular que el nodo Alset de referencia). LabFlow se añade como dominio y app sin mezclar el vertical de ventas.

## Requisitos

- Go 1.22+ (el módulo declara una versión reciente; usa la del `go.mod`)
- Docker y Docker Compose (recomendado)
- Opcional: Supabase (si no, persistencia en disco `alset_data/`)

## Instalación rápida

```bash
git clone https://github.com/yecharlot/AlsetBio.git
cd AlsetBio
cp .env.example .env
docker compose up -d --build
```

Abre: [http://localhost:8080](http://localhost:8080)

### Desarrollo local (sin Docker)

```bash
cp .env.example .env
go mod download
go run ./cmd/prisma-tec
```

## LabFlow + IPFS (en acción)

Las muestras y eventos de custodia se persisten como **bloques content-addressed (CID)** en el blockstore del nodo (`GenerarCID` / `/api/ipfs/*`).

```bash
# Crear muestra
curl -s -X POST http://localhost:8080/api/labflow/samples \
  -H 'Content-Type: application/json' \
  -d '{"type":"blood","org_id":"lab-1","actor":"tech-1","location":"Receiving-A"}'

# Transición de estado
curl -s -X POST http://localhost:8080/api/labflow/samples/<id>/transition \
  -H 'Content-Type: application/json' \
  -d '{"to_status":"ASSIGNED","actor":"tech-1"}'

# Verificar
curl -s http://localhost:8080/api/labflow/verify/<id>
# UI
open http://localhost:8080/w/labflow.app.ans
```

Cada cambio actualiza un **root CID** de índice LabFlow (`GET /api/labflow/root`).

## Configuración

Copia `.env.example` → `.env`. Variables principales:

| Variable | Descripción |
|----------|-------------|
| `PORT` | Puerto HTTP (default 8080) |
| `SUPABASE_URL` | Opcional |
| `SUPABASE_SERVICE_KEY` | Opcional |
| `RENDER` | Si está definida, el nodo actúa como relay (sin cliente de pulsos) |

## Documentación

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — arquitectura del nodo
- [docs/labflow.md](docs/labflow.md) — LabFlow (dominio y roadmap)
- [docs/development.md](docs/development.md) — desarrollo y tests
- [docs/GUIA.md](docs/GUIA.md) — guía de uso del nodo

## Roadmap

| Fase | Contenido |
|------|-----------|
| **1** | LabFlow MVP (muestras, custody, verify, dashboard) |
| **2** | BioPassport |
| **3** | Decision Engine integration |
| **4** | Instrument integrations |
| **5** | Multi-lab / nodos distribuidos |
| **6** | BioResearch ecosystem |

## Licencia

Código de aplicación LabFlow y documentación de AlsetBio: ver `LICENSE`.

El runtime del nodo deriva del diseño Alset/PrismaTec. Estado de licencia del nodo de origen: **verificar en el repositorio de origen**; este repo no reclama derechos exclusivos sobre el ecosistema Alset completo.

## Relación con PrismaTec

| Repo | Uso |
|------|-----|
| [yecharlot/PrismaTec](https://github.com/yecharlot/PrismaTec) | Nodo + Alset Sales Hub |
| **yecharlot/AlsetBio** | Nodo + vertical laboratorio (LabFlow) |

No sincronizar a ciegas cambios de Sales Hub hacia aquí ni al revés.
