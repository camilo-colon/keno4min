# Keno4min

Servicio interno responsable de los tickets y sorteos de Keno4min.

El proyecto se encuentra en su etapa inicial. Todavia no contiene endpoints ni
implementaciones de negocio.

## Inicio rapido

El repositorio incluye un harness de ingenieria para que humanos y agentes
trabajen con el mismo contexto y los mismos criterios de salida:

```bash
./init.sh
```

Ese comando valida la estructura del repositorio, las reglas arquitectonicas
basicas y, cuando haya paquetes Go, ejecuta formato, `go vet` y tests.

El servidor HTTP se ejecuta con `go run ./cmd/api`. Todavia no publica rutas de
producto. Su configuracion usa las siguientes variables de entorno:

| Variable | Valor por defecto |
| --- | --- |
| `KENO4MIN_HTTP_ADDRESS` | `:8080` |
| `KENO4MIN_HTTP_BODY_LIMIT_BYTES` | `4194304` (4 MiB) |
| `KENO4MIN_HTTP_READ_TIMEOUT` | `5s` |
| `KENO4MIN_HTTP_WRITE_TIMEOUT` | `10s` |
| `KENO4MIN_HTTP_IDLE_TIMEOUT` | `60s` |
| `KENO4MIN_HTTP_SHUTDOWN_TIMEOUT` | `10s` |

Los tiempos usan el formato de duracion de Go, por ejemplo `500ms` o `2m`.

Antes de implementar una tarea:

1. Lee [AGENTS.md](AGENTS.md), que funciona como mapa del repositorio.
2. Consulta [ARCHITECTURE.md](ARCHITECTURE.md) y las ADR relacionadas.
3. Para trabajo complejo, crea un plan desde
   [docs/exec-plans/TEMPLATE.md](docs/exec-plans/TEMPLATE.md).
4. Define criterios de aceptacion verificables y termina ejecutando `./init.sh`.

La base de conocimiento esta indexada en [docs/README.md](docs/README.md). Los
criterios usados para revisar un cambio estan en [CHECKPOINTS.md](CHECKPOINTS.md).

## Agentes de Codex

Los agentes del proyecto viven en `.codex/agents/`. El chat principal delega la
coordinacion a `leader`; la implementacion, la exploracion y la revision quedan
separadas y dejan evidencia en disco. Consulta el protocolo y las reglas de
propiedad en [docs/AGENT_WORKFLOWS.md](docs/AGENT_WORKFLOWS.md).

## Organizacion

- `cmd/api`: punto de entrada del servidor HTTP.
- `internal/server/httpapi`: transporte HTTP y adaptacion de identidad enviada por el gateway.
- `internal/ticket`: reglas y casos de uso de tickets.
- `internal/draw`: reglas y consultas de sorteos.
- `internal/mongodb`: adaptadores de persistencia MongoDB.
- `internal/config`: carga y validacion de configuracion.
- `api`: contrato OpenAPI.
- `migrations/mongodb`: cambios versionados de datos e indices.
- `docs/adr`: decisiones de arquitectura.
- `docs/exec-plans`: planes activos y resultados historicos.
- `docs/agent-reports`: evidencia de trabajo delegado organizada por tarea.
- `.codex/agents`: agentes personalizados descubiertos por Codex.
- `scripts/harness`: verificaciones mecanicas compartidas por local y CI.

Las carpetas se mantienen intencionalmente vacias hasta que exista una
implementacion concreta. Los archivos `.gitkeep` permiten conservar la
estructura inicial en Git y deben eliminarse al agregar archivos reales.
