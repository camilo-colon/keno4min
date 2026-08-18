# Informe de implementacion: servidor Fiber

Estado: `done`

Fecha: 2026-08-18

## Alcance completado

- Se fijo Fiber v3.1.0 y se dejaron sus dependencias resueltas por `go mod tidy`.
- Se implemento configuracion por entorno validada, sin estado global mutable,
  con los defaults aprobados de direccion, limite de cuerpo y timeouts.
- Se construyo un borde Fiber sin rutas de produccion, con limites y timeouts
  aplicados, request ID siempre local, Recover temprano y un `ErrorHandler`
  central `application/problem+json`.
- El manejador conserva el status de `*fiber.Error`, usa codigos estables y no
  serializa el valor de panics ni errores internos.
- `cmd/api` ensambla configuracion, listener, contexto de senales y apagado
  ordenado, sin reglas de negocio.
- Se documentaron las variables de entorno necesarias para ejecutar el proceso.
- Se eliminaron `.gitkeep` solamente de `cmd/api` e `internal/config`, donde se
  agregaron archivos reales.

No se agregaron rutas `/v1`, health, readiness o ping; tampoco identidad,
confianza en proxy headers, logging de payload, reglas de tickets/sorteos ni
persistencia.

## Archivos tocados

- `go.mod`, `go.sum`
- `cmd/api/main.go`, eliminacion de `cmd/api/.gitkeep`
- `internal/config/config.go`, `internal/config/config_test.go`, eliminacion de
  `internal/config/.gitkeep`
- `internal/server/httpapi/server.go`
- `internal/server/httpapi/request_id.go`
- `internal/server/httpapi/problem.go`
- `internal/server/httpapi/server_test.go`
- `README.md`
- Este informe.

Se preservaron `AGENTS.md`, el ExecPlan y los informes ajenos preexistentes.

## Decisiones de implementacion

- Variables de entorno: `KENO4MIN_HTTP_ADDRESS`,
  `KENO4MIN_HTTP_BODY_LIMIT_BYTES`, `KENO4MIN_HTTP_READ_TIMEOUT`,
  `KENO4MIN_HTTP_WRITE_TIMEOUT`, `KENO4MIN_HTTP_IDLE_TIMEOUT` y
  `KENO4MIN_HTTP_SHUTDOWN_TIMEOUT`. Las duraciones usan `time.ParseDuration`.
- La direccion debe ser `host:port` con puerto entre 1 y 65535; limites y
  duraciones deben ser positivos.
- `TrustProxy` permanece deshabilitado y no se configura `ProxyHeader`.
- El request ID se genera con `crypto/rand.Text`, se guarda solo durante la
  solicitud y sustituye cualquier `X-Request-ID` entrante. Se expone en la
  cabecera de respuesta y en Problem Details.
- Problem Details usa `about:blank`, titulo HTTP controlado, codigo estable,
  detalle seguro, path como `instance` y request ID. Para errores no reconocidos
  se responde 500 sin incluir `err.Error()`.
- Fiber gestiona el shutdown al cancelar el contexto mediante
  `GracefulContext` y el presupuesto validado de `ShutdownTimeout`.
- Las unicas rutas registradas viven en tests como fixtures efimeros para 405,
  panic, body limit y lifecycle.

## Pruebas

Las pruebas de configuracion cubren defaults, override completo y entradas
invalidas de direccion, limite y cada timeout. Las pruebas del borde cubren 404,
405 con `Allow`, preservacion de status de `*fiber.Error`, panic 500 sin secreto,
media type, reemplazo de request ID entrante, aplicacion de limite 413, wiring de
limites/timeouts y apagado por contexto.

`app.Test` cubre 404, 405, panic, errores Fiber, media type y request ID. Fiber
v3 devuelve `fasthttp.ErrBodyTooLarge` directamente desde `app.Test` antes de
crear una respuesta; por eso el 413 se verifica con el listener en memoria y el
borde real, siguiendo el patron de integracion del propio proyecto Fiber.

## Verificacion

| Comando | Resultado |
| --- | --- |
| `./init.sh` (linea base) | PASS; aun no habia paquetes Go |
| `GOCACHE=/tmp/keno4min-go-build go test ./internal/config ./internal/server/httpapi ./cmd/api` | PASS tras adaptar las pruebas de socket y body limit al listener en memoria |
| `GOCACHE=/tmp/keno4min-go-vet go vet ./...` | PASS |
| `GOCACHE=/tmp/keno4min-go-test go test ./...` | PASS |
| `GOCACHE=/tmp/keno4min-go-race go test -race ./...` | PASS |
| `./init.sh` (final) | PASS; harness verde |
| `git diff --check` | PASS |
| `GOCACHE=/tmp/keno4min-go-mod go mod tidy -diff` | PASS, sin diff |
| busqueda de registro de rutas fuera de tests | PASS, ninguna ruta de produccion |

Dos intentos iniciales fallaron por restricciones del sandbox y no por el
codigo: el cache Go predeterminado no era escribible, por lo que se uso
`GOCACHE` bajo `/tmp`; y `net.Listen` no podia abrir un socket, por lo que las
pruebas de integracion usan `fasthttputil.NewInmemoryListener`.

## Pruebas omitidas y riesgo residual

- No se ejecuto una prueba de proceso enviando `SIGTERM` real ni se abrio un
  puerto del sistema debido a la restriccion de sockets del sandbox. El
  lifecycle equivalente se probo cancelando el contexto conectado a Fiber; el
  wiring de `signal.NotifyContext` queda cubierto por la biblioteca estandar.
- No se verificaron timeouts contra clientes lentos reales. Se comprobo que los
  valores validados llegan exactamente a `fiber.Config`; su semantica depende de
  Fiber v3.1.0.
- Context7 no estuvo disponible en esta sesion. La implementacion se baso en los
  informes previos que consultaron documentacion y codigo oficiales de Fiber;
  no se atribuye investigacion a Context7.
