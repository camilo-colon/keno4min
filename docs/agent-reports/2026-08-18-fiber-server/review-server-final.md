Verdict: APPROVED

# Revision independiente final: servidor HTTP Fiber

Fecha: 2026-08-18

## Alcance y fuentes

Se reviso de nuevo el estado real completo, incluidos archivos rastreados y no
rastreados, contra los criterios de aceptacion de
`docs/exec-plans/active/2026-08-18-fiber-server.md`. Se inspeccionaron
`AGENTS.md`, `ARCHITECTURE.md`, `docs/PRODUCT.md`, `docs/ENGINEERING.md`,
`docs/VERIFICATION.md`, `CHECKPOINTS.md`, ADR 0001/0002/0003, OpenAPI v1,
informes previos y todo el codigo, pruebas, dependencias y documentacion de la
tarea. El cambio preexistente de `AGENTS.md` se preservo como ajeno y no forma
parte de esta revision.

## Hallazgos

No hay hallazgos bloqueantes ni accionables pendientes.

## Cierre de los hallazgos anteriores

### Lifecycle, drenaje y errores — Resuelto

- `internal/server/httpapi/server.go:60-95` ya no usa
  `fiber.ListenConfig.GracefulContext`. Inicia el listener en un serve loop
  propio, sincroniza readiness despues de que fasthttp registra el listener,
  espera sincronicamente `ShutdownWithTimeout` y espera tambien el resultado del
  serve loop.
- Los errores de shutdown y listener se envuelven y conservan con
  `errors.Join`; un deadline de apagado llega al composition root mediante
  `errors.Is(err, context.DeadlineExceeded)`.
- Un contexto cancelado antes de iniciar cierra el listener sin arrancar Fiber,
  eliminando la carrera de cancelacion previa.
- `internal/server/httpapi/server_test.go:158-245` demuestra con el borde real
  en memoria: request activa drenada antes de retornar, timeout propagado,
  cancelacion previa sin listener vivo y preservacion simultanea de errores de
  shutdown/listener. Las pruebas liberan los handlers bloqueados y usan canales
  bufferizados, sin fugas observadas por race o repeticion.

### Cabecera `X-Request-ID` saliente — Resuelto

- `internal/server/httpapi/request_id.go:34-36` guarda el ID exclusivamente en
  locals; no escribe cabecera de respuesta.
- La unica aparicion de `X-Request-ID` en Go es el valor entrante del fixture de
  `TestAppIgnoresIncomingRequestID`, que confirma que el borde genera y usa su
  propio `requestId` en Problem Details.
- La generacion productiva permanece en `crypto/rand.Text`, con al menos 128
  bits de aleatoriedad y probabilidad de colision despreciable. No se fija un
  contrato de propagacion antes de decidir el limite de confianza con gateway.

### Decision transversal Fiber — Resuelto

- `docs/adr/0003-fiber-http-server.md` esta aceptada y documenta alcance,
  ownership, restricciones de `fiber.Ctx`/fasthttp, errores, limites,
  actualizaciones, shutdown y alternativas.
- `docs/README.md:30-37` indexa la ADR. La decision no autoriza endpoints ni
  confianza en headers y mantiene OpenAPI como requisito previo.

## Criterios originales verificados

- `cmd/api` se limita a carga de configuracion, señales, listener, wiring y
  propagacion de errores; no contiene reglas de negocio ni handlers.
- `internal/config` carga defaults/entorno sin estado global mutable y valida
  direccion, limite de cuerpo y todos los timeouts positivos usados por Fiber.
- El borde instala request ID local y Recover antes de handlers productivos. El
  manejador central conserva status de errores Fiber, responde
  `application/problem+json`, incluye campos RFC 9457 y extensiones aprobadas,
  preserva `Allow` para 405 y no serializa panics ni errores internos.
- 404, 405, 413 por listener real, panic, reemplazo del ID entrante, limites y
  timeouts tienen pruebas con la aplicacion Fiber real.
- No hay rutas productivas u operativas fuera de tests. `api/openapi/v1` conserva
  `paths: {}`; no se introdujeron identidad, tickets, sorteos, persistencia o
  idempotencia.
- Fiber v3.1.0 y fasthttp v1.69.0 estan fijados; `go.mod`/`go.sum` estan tidy. Los
  `.gitkeep` eliminados corresponden solo a carpetas que ahora contienen codigo.
- README documenta ejecucion, variables, unidades y defaults reales.

## Validaciones ejecutadas

| Comando | Resultado |
| --- | --- |
| `git status --short` y diff completo excluyendo `AGENTS.md` ajeno | PASS; rastreados y no rastreados inspeccionados, sin artefactos temporales ni alcance extra. |
| `GOCACHE=/tmp/keno4min-review-final-init-cache ./init.sh` | PASS; harness verde. |
| `GOCACHE=/tmp/keno4min-review-final-race2-cache go test -race -count=10 ./...` | PASS; todos los paquetes. |
| `GOCACHE=/tmp/keno4min-review-final-stress2-cache go test -count=30 ./internal/server/httpapi ./internal/config ./cmd/api` | PASS; sin flakes. |
| `GOCACHE=/tmp/keno4min-review-final-vet-cache go vet ./...` | PASS. |
| `GOCACHE=/tmp/keno4min-review-final-tidy-cache go mod tidy -diff` | PASS; sin diff. |
| `git diff --check` | PASS. |
| `rg` de rutas, `GracefulContext` y `X-Request-ID` | PASS; rutas solo en tests, sin `GracefulContext`, sin cabecera saliente. |

## Riesgo residual

No se envio una señal real a un proceso ni se abrio un socket TCP del sistema
por las restricciones del sandbox. Los contextos cancelables y listeners en
memoria ejercitan el mismo ownership de lifecycle, incluidas solicitudes
activas y deadline. Read/write/idle timeout se verificaron como wiring a Fiber,
no mediante clientes TCP lentos. Tras vencer el shutdown, un handler Go que no
coopera puede continuar hasta que el proceso salga; ADR 0003 registra que los
handlers futuros deben respetar contextos y presupuestos de I/O. Ninguno de
estos riesgos residuales impide aceptar la infraestructura actual.
