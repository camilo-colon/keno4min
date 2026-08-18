# Servidor HTTP operativo con Fiber v3

Estado: completed
Creado: 2026-08-18
Responsable: leader

## Proposito

El proceso `cmd/api` puede arrancar y detener un servidor HTTP basado en Fiber
v3 con limites configurables y manejo seguro de fallos, sin introducir todavia
recursos de tickets, sorteos, identidad ni persistencia. Pruebas automatizadas
demuestran el borde HTTP operativo y el ciclo de vida; la investigacion y las
decisiones quedan trazadas en informes persistentes.

## Contexto

La linea base del 2026-08-18 contiene solo el harness y el contrato OpenAPI v1
sin operaciones. `./init.sh` termina con codigo 0 y omite paquetes Go porque aun
no existen. `AGENTS.md` ya estaba modificado antes de esta tarea y se preserva.

Fuentes aplicables: `ARCHITECTURE.md`, `docs/PRODUCT.md`,
`docs/ENGINEERING.md`, `docs/VERIFICATION.md`, `CHECKPOINTS.md`, ADR 0001 y ADR
0002. El usuario solicito Context7, pero no hay servidor, herramienta o recurso
Context7 expuesto en esta sesion; se investigara documentacion primaria oficial
de Fiber y se registrara el fallback sin atribuirle resultados a Context7.

## Criterios de aceptacion

- [x] `cmd/api` ensambla y ejecuta un servidor Fiber v3 sin reglas de negocio.
- [x] La configuracion del proceso valida direccion, limites y tiempos usados por
  el servidor, sin estado global mutable.
- [x] El borde recupera panics antes de responder, usa un manejador central de
  errores `application/problem+json` compatible con RFC 9457 y no filtra detalles
  internos.
- [x] Cada respuesta de error incluye un identificador de solicitud generado por
  el servicio; no se confia en identificadores arbitrarios recibidos hasta que el
  limite de confianza con el gateway se defina.
- [x] El servidor aplica limites de cuerpo y tiempos razonables/configurables, y
  el proceso implementa apagado ordenado ante senales.
- [x] Pruebas con el borde real de Fiber verifican al menos 404, panic, ID de
  solicitud, tipo de contenido y limites relevantes; 405 se prueba con una ruta
  registrada solo en el fixture, sin publicar un endpoint de producto. El ciclo
  de vida/config se prueba proporcionalmente al riesgo.
- [x] No se agregan endpoints de negocio ni se inventan identidad, tickets,
  sorteos, MongoDB o idempotencia. OpenAPI permanece sin operaciones si solo se
  expone infraestructura operativa/no versionada.
- [x] `gofmt`, `go vet`, `go test ./...` y `./init.sh` finalizan en verde; una
  revision independiente emite `APPROVED`.

## Alcance y fuera de alcance

Incluye la infraestructura minima de `internal/config`, el adaptador de servidor
HTTP/Fiber, el composition root y sus pruebas/documentacion tecnica estrictamente
necesaria. Excluye endpoints `/v1`, health/readiness si no son necesarios para
probar el servidor, autenticacion/autorizacion, logging de negocio, metricas,
tracing, persistencia y reglas de tickets/sorteos.

## Plan

1. Investigar el repositorio y las practicas oficiales actuales de Fiber v3.
2. Delegar una implementacion minima basada en evidencia, con pruebas.
3. Solicitar revision independiente y corregir hallazgos si los hay.
4. Ejecutar la verificacion final, cerrar y mover este plan.

## Progress

- [2026-08-18 00:00 COT] Linea base establecida: `git status --short` muestra
  solo ` M AGENTS.md` ajeno; `./init.sh` termina con codigo 0.
- [2026-08-18 00:01 COT] Context7 no esta disponible en las herramientas o
  recursos expuestos; se adopta documentacion primaria oficial como fallback.
- [2026-08-18 00:02 COT] Exploraciones completadas:
  [Fiber oficial](../../agent-reports/2026-08-18-fiber-server/exploration-fiber-official.md)
  y [repositorio](../../agent-reports/2026-08-18-fiber-server/exploration-repository.md).
- [2026-08-18 00:03 COT] Primera implementacion completada en
  [implementation-server.md](../../agent-reports/2026-08-18-fiber-server/implementation-server.md).
- [2026-08-18 00:04 COT] Revision independiente solicito cambios en
  [review-server.md](../../agent-reports/2026-08-18-fiber-server/review-server.md):
  esperar y propagar el apagado, no publicar `X-Request-ID` y registrar Fiber en
  una ADR. Correccion delegada al mismo propietario del codigo.
- [2026-08-18 00:05 COT] Correcciones completadas en
  [implementation-fixes.md](../../agent-reports/2026-08-18-fiber-server/implementation-fixes.md):
  lifecycle poseido por el servidor, request ID sin cabecera publica y ADR 0003.
- [2026-08-18 00:06 COT] Revision final
  [APPROVED](../../agent-reports/2026-08-18-fiber-server/review-server-final.md)
  y verificacion final del lider en verde; plan cerrado.

## Surprises & discoveries

- El repositorio declara Go 1.26.0 y aun no contiene paquetes compilables.
- Fiber delega parte del graceful shutdown a una goroutine cuando se usa
  `GracefulContext`; para esperar el drenaje y propagar sus errores, el servidor
  posee explicitamente el serve loop y `ShutdownWithTimeout`.

## Decision log

- [2026-08-18] No se agrega ningun endpoint `/v1`: el contrato no define
  recursos y el usuario pidio infraestructura del servidor, no reglas de
  producto.
- [2026-08-18] La investigacion usa fuentes oficiales de Fiber porque Context7
  no esta disponible; esta limitacion se mantiene visible.
- [2026-08-18] Los defaults locales y reversibles seran direccion `:8080`, cuerpo
  de 4 MiB, lectura 5 s, escritura 10 s, inactividad 60 s y apagado 10 s; cada
  valor sera configurable por entorno y validado antes de arrancar. Estos son
  presupuestos iniciales operativos, no garantias del contrato ni reglas de
  producto.
- [2026-08-18] El servicio siempre genera su propio request ID y no propaga el
  valor entrante hasta que el limite de confianza con el gateway sea definido.
- [2026-08-18] No se publica health, readiness, ping ni otra ruta. Los fixtures
  pueden registrar handlers efimeros para comprobar 405, panic y body limit sin
  ampliar la superficie del binario.

## Validation

| Comando o prueba | Resultado | Fecha |
| --- | --- | --- |
| `git status --short` | ` M AGENTS.md` preexistente | 2026-08-18 |
| `./init.sh` | PASS; no hay paquetes Go | 2026-08-18 |
| `GOCACHE=/tmp/keno4min-leader-vet go vet ./...` | PASS | 2026-08-18 |
| `GOCACHE=/tmp/keno4min-leader-race go test -race ./...` | PASS | 2026-08-18 |
| `GOCACHE=/tmp/keno4min-leader-tidy go mod tidy -diff` | PASS; sin diff | 2026-08-18 |
| `git diff --check` | PASS | 2026-08-18 |
| `GOCACHE=/tmp/keno4min-leader-init ./init.sh` final | PASS; harness verde | 2026-08-18 |

## Recuperacion

Los cambios son locales y no crean datos externos. Cada fase puede reintentarse
desde el informe del agente correspondiente. Una reversion eventual debe
limitarse a los archivos creados por esta tarea y preservar `AGENTS.md`.

## Outcome

Se incorporo Fiber v3.1.0 como borde HTTP transversal, con configuracion
validada, Problem Details seguro, request ID local, limites y lifecycle con
apagado esperado y errores propagados. No se publicaron rutas; OpenAPI conserva
`paths: {}`. La decision estable quedo en ADR 0003 y las practicas/limitaciones
de Fiber en los informes de exploracion.

La revision final fue `APPROVED`. No se probaron senales reales, sockets TCP ni
clientes lentos por restricciones del sandbox; listeners en memoria cubrieron
drenaje, timeout, errores y cancelacion. Context7 no estuvo disponible y la
investigacion uso exclusivamente fuentes oficiales de Fiber como fallback.
