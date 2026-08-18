# Informe de correcciones: servidor Fiber

Estado: `done`

Fecha: 2026-08-18

## Alcance corregido

Se atendieron todos los hallazgos de
`review-server.md` sin modificar OpenAPI, producto, negocio, persistencia,
ExecPlan ni informes ajenos.

1. `Server.Serve` posee ahora explicitamente el lifecycle. Arranca Fiber sin
   `GracefulContext`, sincroniza readiness desde la primera llamada a `Accept`
   (despues de que fasthttp registra el listener), espera
   `ShutdownWithTimeout`, espera tambien el serve loop y combina los errores de
   ambas operaciones. Un contexto cancelado antes del arranque cierra el
   listener sin iniciar una goroutine.
2. El request ID sigue siendo generado localmente e ignora cualquier valor
   entrante, pero ya no se publica en `X-Request-ID`. Solo permanece en locals
   durante la solicitud y en `requestId` de Problem Details.
3. Se agrego e indexo la ADR aceptada 0003 para Fiber v3, incluyendo alcance,
   alternativas, restricciones de fasthttp/`fiber.Ctx`, actualizaciones,
   ownership del shutdown, pruebas, limites y ausencia de endpoints.

## Archivos tocados en esta correccion

- `internal/server/httpapi/server.go`
- `internal/server/httpapi/request_id.go`
- `internal/server/httpapi/server_test.go`
- `docs/adr/0003-fiber-http-server.md`
- `docs/README.md`
- Este informe.

Se preservaron `AGENTS.md`, OpenAPI, el ExecPlan, el informe de implementacion
anterior y los informes de exploracion/revision.

## Detalles y pruebas agregadas

- Readiness no usa `BeforeServeFunc`, porque ese callback ocurre antes de que
  fasthttp registre el listener. Un wrapper de `net.Listener` notifica desde
  `Accept`, punto en el que un shutdown ya puede encontrar y cerrar el listener.
- La cancelacion posterior a readiness llama y espera
  `app.ShutdownWithTimeout`; despues espera el resultado del listener. Los
  errores se envuelven con contexto y se unen con `errors.Join`.
- Una prueba mantiene un handler activo, cancela el contexto y demuestra que
  `Serve` no retorna hasta liberar y completar la request dentro del presupuesto.
- Otra mantiene la request mas alla del presupuesto y demuestra
  `errors.Is(err, context.DeadlineExceeded)`; despues libera el handler para no
  dejar goroutines de prueba.
- Una prueba entrega un contexto ya cancelado y confirma que el listener queda
  cerrado sin arrancar Fiber.
- Un listener controlado produce simultaneamente error de cierre y error de
  serve; la prueba confirma que ambos permanecen detectables mediante
  `errors.Is`.
- Los tests de Problem Details conservan una cabecera entrante como estimulo,
  pero solo verifican que `requestId` contiene el ID local; no fijan ninguna
  cabecera de respuesta.

## Verificacion

| Comando | Resultado |
| --- | --- |
| `GOCACHE=/tmp/keno4min-fixes-test go test -count=10 ./internal/server/httpapi` | PASS |
| `GOCACHE=/tmp/keno4min-fixes-race go test -race -count=20 ./internal/server/httpapi` | PASS |
| `GOCACHE=/tmp/keno4min-fixes-directed go test -race -count=20 ./internal/server/httpapi` | PASS, incluida combinacion de errores |
| `GOCACHE=/tmp/keno4min-fixes-vet go vet ./...` | PASS |
| `GOCACHE=/tmp/keno4min-fixes-all go test ./...` | PASS |
| `GOCACHE=/tmp/keno4min-fixes-race-all go test -race ./...` | PASS |
| `./init.sh` | PASS; harness verde e indice de ADR consistente |
| `git diff --check` | PASS |
| `GOCACHE=/tmp/keno4min-fixes-mod go mod tidy -diff` | PASS, sin diff |
| busqueda de rutas fuera de tests | PASS, ninguna ruta de produccion |
| busqueda de `X-Request-ID` | Solo queda el header entrante del test que demuestra que se ignora; no hay escritura de respuesta |

## Pruebas omitidas y riesgo residual

- No se envio `SIGTERM` a un proceso real ni se abrio un socket TCP del sistema
  por las restricciones del sandbox. El ownership completo se probo con
  listeners en memoria y contextos cancelables, incluida request activa y
  timeout.
- Los timeouts de lectura/escritura siguen verificados como wiring hacia Fiber,
  no mediante clientes TCP lentos.
- Tras vencer el shutdown, fasthttp devuelve el error de deadline y el proceso
  puede continuar su salida; Go no puede interrumpir por la fuerza un handler
  arbitrario. La prueba libera el handler despues de observar el error para
  demostrarlo sin fuga. Los handlers futuros deberan respetar contextos y
  presupuestos propios de I/O.
- Context7 no estuvo disponible; la ADR y la implementacion se apoyan en la
  documentacion y codigo oficiales ya investigados y no atribuyen resultados a
  Context7.
