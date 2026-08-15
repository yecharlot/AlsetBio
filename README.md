# AlsetBio

Plataforma para laboratorios: muestras, cadena de custodia, verificación y workflows configurables. El código de ejecución (identidad, almacenamiento por CID, API HTTP, red P2P) proviene del diseño abierto documentado en [PrismaTec](https://github.com/yecharlot/PrismaTec); este repositorio es un producto aparte, orientado solo a uso científico y de laboratorio.

**No es un dispositivo médico.** No diagnostica ni prescribe. Sirve para operar el flujo de muestras y dejar evidencia comprobable.

---

## Arranque en cuatro pasos

```bash
git clone https://github.com/yecharlot/AlsetBio.git
cd AlsetBio
cp .env.example .env
docker compose up -d --build
```

Interfaz LabFlow: [http://localhost:8080/w/labflow.app.ans](http://localhost:8080/w/labflow.app.ans)  
Panel general del runtime: [http://localhost:8080/static/index.html](http://localhost:8080/static/index.html)

Sin Docker:

```bash
cp .env.example .env
go run ./cmd/prisma-tec
```

Variables útiles en `.env`:

| Variable | Uso |
|----------|-----|
| `PORT` | Puerto HTTP (por defecto 8080) |
| `SUPABASE_URL` / `SUPABASE_SERVICE_KEY` | Persistencia en Supabase (si no, disco local `alset_data/`) |
| `LABFLOW_REQUIRE_AUTH` | `true` obliga token Bearer en la API LabFlow |

---

## Qué incluye hoy

### LabFlow (aplicación principal)

Gestión del ciclo de vida de muestras:

1. Alta de muestra (identificador tipo `BIO-2026-000001`)
2. Estados controlados (no se puede saltar el flujo a ciegas)
3. Eventos de custodia (append-only, cada uno con referencia CID)
4. Verificación pública reducida (`/verify/<id>`)
5. Código QR hacia la página de verificación
6. Roles de laboratorio y alcance por organización
7. Workflows intercambiables (`default`, `water-testing`, `clinical`)

UI: `/w/labflow.app.ans`

### Runtime compartido (base técnica)

Misma base que en PrismaTec, usada aquí como motor:

- API HTTP y apps estáticas bajo `/w/<nombre>.app.ans`
- Bloques content-addressed (CID) en disco/`blocks` y endpoints `/api/ipfs/*`
- Agentes, tokens y roles
- Persistencia local o Supabase
- Opcional: red libp2p y pulsos entre instancias

Detalle de arquitectura del motor: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) y el repo de origen [PrismaTec](https://github.com/yecharlot/PrismaTec).

### Otras apps en `static/apps`

| App | Ruta | Estado |
|-----|------|--------|
| **labflow** | `/w/labflow.app.ans` | Operativa (muestras, custodia, QR, stats) |
| **config** | `/w/config.app.ans` | Utilidad de configuración heredada del runtime |

LabFlow es el producto; el resto es soporte del entorno de ejecución.

---

## Cómo publicar una aplicación nueva

Cualquier carpeta con un `index.html` en `static/apps/<nombre>/` queda servida en:

```text
http://localhost:8080/w/<nombre>.app.ans
```

También puedes registrar archivos en caliente:

```bash
curl -X POST http://localhost:8080/api/apps/register \
  -F "appName=mi-app" \
  -F "files=@index.html"
```

Buenas prácticas:

1. Una carpeta por app, nombre en minúsculas sin espacios
2. `index.html` como entrada
3. Llamadas a la API del mismo origen (`/api/...`) para evitar líos de CORS en local
4. No mezclar lógica de ventas u otros verticales ajenos a laboratorio en este repo

---

## API LabFlow (referencia rápida)

```text
GET  /api/labflow/stats
GET  /api/labflow/workflows
GET  /api/labflow/root
GET  /api/labflow/samples
POST /api/labflow/samples
GET  /api/labflow/samples/:id
POST /api/labflow/samples/:id/transition
GET  /api/labflow/samples/:id/events
GET  /api/labflow/verify/:id
GET  /api/labflow/qr/:id
POST /api/labflow/auth/token
GET  /verify/:id
```

### Crear una muestra

```bash
curl -s -X POST http://localhost:8080/api/labflow/samples \
  -H "Content-Type: application/json" \
  -H "X-Lab-Role: TECHNICIAN" \
  -H "X-Lab-Org: lab-1" \
  -d '{"type":"blood","workflow_id":"default","location":"Receiving-A"}'
```

### Avanzar estado

```bash
curl -s -X POST http://localhost:8080/api/labflow/samples/<id>/transition \
  -H "Content-Type: application/json" \
  -H "X-Lab-Role: TECHNICIAN" \
  -H "X-Lab-Org: lab-1" \
  -d '{"to_status":"ASSIGNED"}'
```

### Token (cuando `LABFLOW_REQUIRE_AUTH=true`)

```bash
curl -s -X POST http://localhost:8080/api/labflow/auth/token \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"tech-1","roles":["TECHNICIAN"],"org_id":"lab-1"}'
# Luego: Authorization: Bearer <token>
```

En desarrollo, sin token, bastan las cabeceras `X-Lab-Role`, `X-Lab-Org` y `X-Lab-Actor`.

---

## Roles

| Rol | Alcance típico |
|-----|----------------|
| `LAB_ADMIN` | Administración completa |
| `LAB_MANAGER` | Gestión del laboratorio |
| `TECHNICIAN` | Alta y avance operativo (ASSIGNED, IN_PROGRESS, FLAGGED) |
| `REVIEWER` | QC, RELEASED, ARCHIVED, FLAGGED |
| `CLIENT` | Consulta limitada a sus muestras |

---

## Workflows

| ID | Notas |
|----|--------|
| `default` | Flujo general de laboratorio |
| `water-testing` | Permite reensayo `QC_REVIEW → IN_PROGRESS` |
| `clinical` | Tras `FLAGGED` solo cabe `ARCHIVED` |

La muestra guarda `workflow_id`; las transiciones se validan contra ese flujo.

---

## Persistencia e integridad

- Cada muestra y cada evento de custodia se guarda como bloque con **CID**
- El índice LabFlow también es un bloque; su CID aparece en `GET /api/labflow/root`
- Tras reiniciar el proceso, el índice se recupera desde `alset_data/labflow_root.cid` y el almacén de bloques

No hay blockchain ni criptomoneda: solo direccionamiento por contenido del runtime.

---

## Estructura del repositorio

```text
cmd/prisma-tec/     entrada del proceso
internal/labflow/   dominio de muestras, custodia, roles, workflows
internal/node/      runtime HTTP, red, bloques CID
static/apps/        aplicaciones web (labflow, config, …)
docs/               arquitectura y guías
docker-compose.yml
```

Desarrollo y tests: [docs/development.md](docs/development.md)  
LabFlow en detalle: [docs/labflow.md](docs/labflow.md)

```bash
go test ./internal/labflow/ -count=1
```

---

## Roadmap

| Fase | Contenido |
|------|-----------|
| 1 | LabFlow (en curso: muestras, IPFS/CID, roles, workflows) |
| 2 | BioPassport |
| 3 | Motor de decisión (PASS / REVIEW / FLAG) |
| 4 | Integración con instrumentos |
| 5 | Varias sedes / instancias distribuidas |
| 6 | BioResearch |

---

## Licencia y origen

Ver [LICENSE](LICENSE). El motor de ejecución sigue el modelo descrito en [PrismaTec](https://github.com/yecharlot/PrismaTec). El trabajo de producto de laboratorio (LabFlow y documentación de este repo) se mantiene aquí, separado del vertical comercial de ventas.
