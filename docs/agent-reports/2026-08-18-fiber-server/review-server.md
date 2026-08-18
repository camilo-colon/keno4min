Verdict: CHANGES_REQUESTED

# Revision independiente: servidor HTTP Fiber

Fecha: 2026-08-18

## Alcance revisado

Se contrastaron los criterios de aceptacion del ExecPlan activo con el estado y
contenido real del arbol, incluidos archivos rastreados y no rastreados. Se
leyeron `AGENTS.md`, `ARCHITECTURE.md`, `docs/PRODUCT.md`,
`docs/ENGINEERING.md`, `docs/VERIFICATION.md`, `CHECKPOINTS.md`, ADR 0001/0002,
OpenAPI v1 y los informes de exploracion e implementacion. El cambio
preexistente en `AGENTS.md` se trato como ajeno y no se reviso ni modifico.

## Hallazgos

### Alta — El proceso retorna antes de completar el apagado graceful y descarta sus errores

- **Ubicacion:** `internal/server/httpapi/server.go:58-68`,
  `cmd/api/main.go:58-61`, `internal/server/httpapi/server_test.go:160-185`.
- **Evidencia:** `Server.Serve` entrega el contexto a
  `fiber.ListenConfig.GracefulContext` y retorna directamente el resultado de
  `app.Listener`. En Fiber v3.1.0, `Listener` lanza `gracefulShutdown` en una
  goroutine (`listen.go:269-278`) y retorna `app.server.Serve(ln)`
  (`listen.go:303`). Esa goroutine llama a `ShutdownWithTimeout`
  (`listen.go:560-577`). La implementacion fijada de fasthttp v1.69.0 cierra el
  listener al inicio del shutdown y documenta/implementa que `Serve` retorna
  inmediatamente, mientras `ShutdownWithContext` sigue esperando que
  `open == 0` o que venza el contexto (`server.go:1928-1975`). La propia API de
  Fiber advierte que el programa debe esperar a que `ShutdownWithTimeout`
  retorne (`app.go:1100-1110`). Aqui nadie espera esa llamada: cuando el
  listener se cierra, `Server.Serve`, `run`, `execute` y `main` pueden terminar
  aunque haya una solicitud activa. Ademas, el error de timeout solo llega a
  hooks internos de Fiber y nunca se propaga por `Server.Serve`.
- **Impacto:** un `SIGTERM` puede terminar el proceso y cortar trabajo en vuelo
  pese a que el binario declare un presupuesto de apagado de 10 segundos. Un
  timeout o fallo de shutdown se informa como exito, impidiendo al supervisor y
  a observabilidad distinguir un drenaje limpio de uno forzado. Tambien queda
  una ventana de arranque: `GracefulContext` inicia su goroutine antes de que
  Fiber inicialice el servidor; una cancelacion entre el `ctx.Err()` de
  `cmd/api/main.go:37` y el arranque puede hacer que `ShutdownWithTimeout`
  observe `ErrNotRunning` y que el listener arranque despues sin otra
  cancelacion pendiente.
- **Reproduccion:** registrar un fixture cuyo handler notifique que entro y se
  bloquee; arrancar `Serve`, iniciar la request y cancelar el contexto. Con la
  implementacion actual el listener se cierra y `Serve` puede entregar `nil`
  antes de liberar el handler. Un segundo caso que mantenga el handler mas alla
  de `ShutdownTimeout` demuestra que el caller sigue recibiendo `nil` en lugar
  del error de deadline. La prueba actual solo cancela sin solicitudes activas
  y comprueba el retorno temprano, por lo que no valida drenaje.
- **Correccion requerida:** hacer que el paquete posea explicitamente el ciclo
  de vida: arrancar `Listener` sin `GracefulContext`, sincronizar que Fiber ya
  esta listo, llamar y esperar `app.ShutdownWithTimeout` al cancelar, esperar
  tambien el resultado del serve loop y combinar/propagar los errores sin
  carreras de arranque. Agregar pruebas deterministas para (1) una request
  activa que termina dentro del presupuesto, (2) una request que excede el
  presupuesto y expone el error de shutdown, y (3) contexto cancelado antes o
  durante el arranque sin dejar el servidor ejecutandose.

### Media — La cabecera de correlacion crea un contrato que la fuente de verdad mantiene pendiente

- **Ubicacion:** `internal/server/httpapi/request_id.go:34-37`,
  `internal/server/httpapi/server_test.go:72-85`.
- **Evidencia:** `setRequestID` escribe `X-Request-ID` en todas las respuestas y
  el test fija ese comportamiento. ADR 0002 solo define `requestId` como
  extension de Problem Details. `docs/PRODUCT.md:14-27` mantiene pendiente la
  propagacion del identificador de solicitud y los encabezados aceptados. La
  exploracion de repositorio tambien dejo explicitamente documentado que no
  existe una cabecera de respuesta aprobada (`exploration-repository.md:94-97`).
  El ExecPlan autoriza generar un ID local y no confiar en el entrante, pero no
  decide una cabecera de respuesta para respuestas exitosas o de
  infraestructura.
- **Impacto:** el gateway puede empezar a depender de una interfaz HTTP no
  acordada ni documentada; retirar o renombrar la cabecera despues seria un
  cambio de compatibilidad. La seguridad respecto al valor entrante si esta
  bien implementada: se ignora y se genera `crypto/rand.Text`, con al menos 128
  bits de aleatoriedad y colisiones practicamente despreciables.
- **Reproduccion:** enviar `GET /missing` con cualquier `X-Request-ID`; la
  respuesta 404 siempre incluye otro `X-Request-ID`, aunque OpenAPI y las ADR no
  describen esa cabecera. `TestAppAlwaysReplacesIncomingRequestID` lo confirma.
- **Correccion requerida:** conservar el ID local en `requestId` de Problem
  Details, pero eliminar la cabecera y su asercion hasta que exista una decision
  de propagacion; o decidir y documentar expresamente la cabecera, alcance y
  semantica antes de fijarla en tests. No se requiere cambiar OpenAPI mientras
  no haya operaciones productivas.

### Media — La adopcion transversal de Fiber/fasthttp no tiene la ADR exigida

- **Ubicacion:** `go.mod:5-8`, `internal/server/httpapi/server.go:11-13`,
  `ARCHITECTURE.md:82-86`.
- **Evidencia:** el cambio fija Fiber v3.1.0 y expone `*fiber.App` como base para
  todos los adaptadores HTTP futuros; tambien incorpora semanticas propias de
  fasthttp para limites, contextos, tests y shutdown. `ARCHITECTURE.md` exige una
  ADR al introducir una tecnologia transversal o una decision costosa de
  revertir. Solo existen ADR 0001 (paquetes) y 0002 (convenciones HTTP); el
  ExecPlan no sustituye esa fuente estable.
- **Impacto:** el siguiente flujo vertical no tiene una decision aceptada que
  documente por que Fiber es el adaptador elegido, sus restricciones de
  `fiber.Ctx`/fasthttp, la estrategia de actualizacion ni la responsabilidad del
  lifecycle. El defecto de shutdown anterior muestra que esas diferencias no
  son meramente cosmeticas.
- **Reproduccion:** `git status --short` y `go.mod` muestran la nueva dependencia,
  mientras `docs/adr/` e indice solo contienen 0001/0002.
- **Correccion requerida:** agregar e indexar una ADR aceptada para Fiber v3 que
  registre al menos alcance, alternativas relevantes, consecuencias de
  fasthttp/contexto, politica de versionado y ownership del apagado. No hace
  falta agregar una operacion OpenAPI: el diff no registra rutas productivas y
  `paths` debe seguir vacio.

## Aspectos verificados sin hallazgo

- `cmd/api` solo carga configuracion, crea listener, ensambla el servidor y
  maneja ciclo de proceso; no contiene reglas de negocio ni rutas.
- La configuracion documenta y valida direccion, limite positivo y duraciones
  positivas; los valores llegan a `fiber.Config`, `TrustProxy` permanece
  deshabilitado y no hay estado global mutable.
- El orden request-ID/Recover garantiza ID local para panics de handlers; el ID
  entrante no se reutiliza y `crypto/rand.Text` ofrece aleatoriedad suficiente.
- 404, 405 con `Allow`, 413 mediante listener real en memoria y panic 500 pasan
  por el manejador central. Las respuestas observadas son JSON con
  `application/problem+json`, status, codigo, instance controlada y request ID;
  los detalles internos no se serializan. El `json.Marshal` actual solo recibe
  strings e `int`, por lo que no tiene una ruta de error alcanzable; si el DTO se
  amplia con tipos falibles, debera agregarse un fallback seguro probado.
- No hay rutas productivas, cambios de OpenAPI, identidad, negocio o
  persistencia. La eliminacion de `.gitkeep` esta limitada a carpetas que ahora
  contienen Go. `go.mod` y `go.sum` estan tidy.

## Validaciones ejecutadas

| Comando | Resultado |
| --- | --- |
| `git status --short` | Cambio ajeno `AGENTS.md` preservado; diff y untracked de la tarea inspeccionados. |
| `GOCACHE=/tmp/keno4min-review-init-cache ./init.sh` | PASS; harness verde. |
| `GOCACHE=/tmp/keno4min-review-race-cache go test -race -count=10 ./...` | PASS. |
| `GOCACHE=/tmp/keno4min-review-test-cache go test -count=50 ./internal/config ./internal/server/httpapi ./cmd/api` | PASS; no se observaron flakes. |
| `GOCACHE=/tmp/keno4min-review-mod-cache go mod tidy -diff` | PASS; sin diff. |
| `git diff --check` | PASS. |
| Busqueda de registro de rutas fuera de tests con `rg` | PASS; ninguna ruta productiva. |
| Inspeccion de Fiber v3.1.0 y fasthttp v1.69.0 en el module cache | Confirma que `Serve` retorna al cerrar listener antes de que el shutdown termine. |

## Riesgo residual

No se abrio un socket TCP del sistema ni se envio una señal real debido al
entorno sandbox; el listener en memoria cubre el borde Fiber pero no sustituye
una prueba de proceso. Los timeouts de lectura/escritura se verificaron como
wiring, no con clientes lentos. Esos riesgos son aceptables despues de corregir
el lifecycle y dejar evidencia de drenaje/timeout; hoy el defecto confirmado
impide aprobar.
