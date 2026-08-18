# Exploración: prácticas de Fiber v3 (fuentes oficiales)

## Alcance y limitación

Consulta realizada el 2026-08-18. Este informe cubre solamente comportamiento y
recomendaciones de Fiber v3, contrastados contra documentación oficial de
Fiber (`docs.gofiber.io`) y su repositorio oficial. No se inspeccionó ni modificó
código del servicio.

Se solicitó Context7, pero ese MCP no está expuesto en esta sesión. Por ello no
fue posible consultarlo. El reemplazo usado fueron las fuentes oficiales
enlazadas abajo; ninguna conclusión se atribuye a Context7.

## Evidencia documentada por Fiber

### Límites y timeouts

Fuente: [API Fiber: Config](https://docs.gofiber.io/api/fiber/) (consultada el
2026-08-18).

* `BodyLimit` limita el tamaño máximo del cuerpo; su valor por defecto es 4 MiB
  y los cuerpos mayores reciben 413. También alcanza cuerpos comprimidos cuando
  se usa `Ctx.Body()`, multipart y el adaptador `net/http`.
* `ReadTimeout` cubre la lectura de la petición completa, incluido el cuerpo;
  su valor por defecto es ilimitado.
* `WriteTimeout` limita la duración de la escritura de la respuesta; por defecto
  es ilimitado.
* `IdleTimeout` limita la espera de la siguiente petición en una conexión
  keep-alive. Si vale cero, Fiber usa `ReadTimeout`.

Inferencia/recomendación: declarar explícitamente los cuatro valores según los
límites publicados de payload y los presupuestos de latencia del producto. No
hay en Fiber un conjunto universal de duraciones seguro: elegirlas sin conocer
proxies, clientes, cargas y operaciones del servicio podría cortar peticiones
legítimas o dejar expuestas conexiones lentas.

### Recuperación, errores y correlación

Fuentes: [Error handling](https://docs.gofiber.io/guide/error-handling/),
[Recover](https://docs.gofiber.io/middleware/recover/) y
[RequestID](https://docs.gofiber.io/middleware/requestid/) (consultadas el
2026-08-18).

* Fiber procesa centralmente los errores devueltos por handlers/middleware. El
  `ErrorHandler` se configura en `fiber.Config` al construir la aplicación.
* El manejador predeterminado convierte un `*fiber.Error` a su código y mensaje;
  para otros errores usa 500 y envía `err.Error()` como texto. Por tanto, Fiber
  documenta que el manejador se puede sustituir por uno propio.
* Fiber no recupera `panic` por defecto. `recover.New()` intercepta panics de
  handlers posteriores y los pasa al `ErrorHandler` central. La configuración
  de Recover tiene `EnableStackTrace: false` por defecto y permite sustituir su
  `PanicHandler`.
* `requestid.New()` genera o propaga el identificador de solicitud, lo agrega a
  cabecera de respuesta y contexto. Por defecto usa `X-Request-ID` y
  `utils.SecureToken`; IDs entrantes fuera del rango ASCII visible se rechazan y
  se regeneran. `requestid.FromContext` lo recupera desde `fiber.Ctx`,
  `context.Context` u otros contextos admitidos.

Inferencia/recomendación: instalar `RequestID` y `Recover` al inicio de la
cadena y hacer que el `ErrorHandler` sea el único traductor de errores a la
forma de error del API. El handler debería registrar internamente el error
original y responder una representación controlada; exponer errores genéricos
tal como hace el predeterminado puede filtrar detalles de dependencias. Esta
forma concreta de respuesta (campos, mensajes, exposición del request ID) debe
derivarse del contrato OpenAPI del repositorio, no de Fiber.

### Pruebas con `app.Test`

Fuente: [API App: Test](https://docs.gofiber.io/api/app/) (consultada el
2026-08-18).

* `app.Test(req, config...)` recibe un `*http.Request` y devuelve
  `*http.Response`, sin que el ejemplo documentado inicie un listener real.
  Se presenta para tests `_test.go` y depuración de routing.
* Sin config, su `TestConfig` efectivo tiene `Timeout: 1s` y
  `FailOnTimeout: true`.
* Pasar `fiber.TestConfig{}` no equivale a omitir la config: produce timeout 0
  y `FailOnTimeout: false`, es decir, un test sin timeout.

Inferencia/recomendación: cubrir middleware, routing, serialización y la forma
de los errores mediante `app.Test`, sin aceptar a ciegas el timeout por defecto
en pruebas que verifican operaciones lentas. Las pruebas de señal, socket,
proxy o timeouts a nivel de conexión necesitan un nivel de integración distinto;
`app.Test` no demuestra esos comportamientos de transporte.

### Lifecycle y apagado

Fuente: [API Fiber: ListenConfig y Server shutdown](https://docs.gofiber.io/api/fiber/)
(consultada el 2026-08-18).

* `ListenConfig.GracefulContext` permite iniciar apagado mediante un contexto.
  Su `ShutdownTimeout` predeterminado es 10 segundos; cero desactiva el límite y
  espera indefinidamente.
* `Shutdown()` cierra listeners y espera indefinidamente a que las conexiones
  activas queden ociosas. `ShutdownWithTimeout` fuerza el cierre de conexiones
  activas al vencer el timeout. `ShutdownWithContext` también puede forzar el
  cierre al vencer el deadline; los hooks de apagado se ejecutan aun con error.
* El repositorio oficial muestra que `Listener` inicia una goroutine de apagado
  cuando se proporciona `GracefulContext`, y llama a `ShutdownWithTimeout` o
  `Shutdown` según `ShutdownTimeout`.

Inferencia/recomendación: el ensamblaje del binario debe poseer la señal del SO
y producir un contexto cancelable/deadline explícito para el ciclo de vida. El
presupuesto de apagado debe coordinarse con el orquestador/proxy y con el tiempo
máximo de trabajo de dependencias; Fiber no determina esos valores externos.

### Riesgos de `fasthttp` / `fiber.Ctx`

Fuentes: [README del repositorio oficial](https://github.com/gofiber/fiber) y
[Go Context](https://docs.gofiber.io/guide/go-context/) (consultadas el
2026-08-18).

* Fiber está construido sobre `fasthttp`.
* La documentación oficial declara que los valores obtenidos de `fiber.Ctx` se
  reutilizan entre solicitudes y son válidos sólo durante el handler. No se debe
  conservar `c` ni usarlo en goroutines que sobrevivan al handler.
* Aunque `fiber.Ctx` implementa `context.Context`, no puede cancelarse:
  `Deadline`, `Done` y `Err` no reflejan cancelación. Para trabajo asíncrono se
  debe obtener `c.Context()` antes de retornar; la documentación también indica
  derivar `WithTimeout`/`WithCancel` desde `c.Context()`, no desde `c`.

Inferencia/recomendación: los puertos de aplicación no deberían aceptar ni
retener `fiber.Ctx`/tipos `fasthttp`; handlers deben extraer datos y pasar tipos
propios o un `context.Context` apropiado a los casos de uso. Esto reduce el
acoplamiento y evita referencias recicladas. Si se requiere cancelación real
por cliente desconectado, el comportamiento exacto debe verificarse contra la
versión de Fiber elegida y mediante prueba de integración.

## Riesgos y decisiones pendientes

1. La versión exacta de Fiber a fijar no está determinada por esta investigación;
   las APIs citadas son v3.x actual. La compatibilidad debe confirmarse con el
   `go.mod` que llegue a aprobarse.
2. Los valores concretos de `BodyLimit` y timeouts, y el presupuesto de
   `ShutdownTimeout`, dependen de contrato, infraestructura y operaciones; no
   se deben inventar.
3. Definir si el API permite al cliente propagar `X-Request-ID` o debe siempre
   reemplazarlo es una política de trazabilidad/seguridad aún no establecida por
   el repositorio.
4. La taxonomía de errores, cuerpo JSON y visibilidad del identificador de
   solicitud deben provenir del OpenAPI, no de ejemplos de framework.
5. Si hay proxy inverso, `TrustProxy`/`TrustProxyConfig` requieren la lista
   real de IPs/CIDRs confiables: Fiber advierte que sin ella las cabeceras son
   suplantables.

## Partición segura del trabajo posterior

1. **Contrato y decisiones (propietario API/producto):** decidir contrato de
   error, límites, timeouts, señal/apagado e infraestructura proxy; reflejar lo
   observable en OpenAPI y ADR si corresponde.
2. **Composición HTTP (propietario `cmd/api`/adaptador):** crear la aplicación
   Fiber, configuración aprobada, middleware en orden decidido, registro de
   rutas y lifecycle. No introducir reglas de negocio allí.
3. **Adaptadores de endpoints (propietario HTTP):** mapear DTO/OpenAPI a los
   puertos de casos de uso, sin filtrar `fiber.Ctx` a la capa de aplicación.
4. **Pruebas (propietario tests):** usar `app.Test` para rutas, middleware y
   errores; añadir integración aislada para listener, señales y timeout si las
   decisiones aprobadas lo requieren. Evitar editar simultáneamente los mismos
   archivos de composición y tests.
