# Exploración del diseño de servidor Fiber v3

## Alcance de la exploración

Se examinó únicamente la evidencia del repositorio aplicable al servidor HTTP:
`AGENTS.md`, el ExecPlan activo, arquitectura, producto, ingeniería,
verificación, checkpoints, ADR 0001/0002, harness, `go.mod`, contrato OpenAPI
y el árbol actual. No se modificó código, contrato, configuración ni pruebas.

`git status --short` al inicio mostró ` M AGENTS.md` (cambio ajeno que debe
preservarse) y `?? docs/exec-plans/active/2026-08-18-fiber-server.md` (plan del
líder). `./init.sh` terminó correctamente el 2026-08-18: no hay paquetes Go,
por lo que el harness omitió `gofmt`, `go vet` y `go test`.

Context7 no está expuesto como herramienta, recurso ni servidor MCP en esta
sesión. El ExecPlan ya registra esa limitación y establece como fallback la
documentación primaria oficial de Fiber; esta exploración no atribuye hallazgos
externos a Context7.

## Evidencia y flujo real

| Fuente / símbolo | Hecho establecido | Consecuencia para el servidor |
| --- | --- | --- |
| `go.mod` | Módulo `cronos.bet/keno4min`, Go `1.26.0`; sin dependencias. | Fiber debe añadirse como dependencia explícita y compatible; no hay biblioteca HTTP ya elegida en el código. |
| `cmd/api/.gitkeep` | No existe `main` ni ensamblaje ejecutable. | `cmd/api` es el punto donde se crea configuración, servidor y ciclo de vida, no reglas de negocio. |
| `internal/config/.gitkeep` | No existe lectura ni validación de configuración. | La configuración del proceso es una responsabilidad concreta aún vacía. |
| `ARCHITECTURE.md`, tabla «Paquetes y responsabilidades» | `cmd/api` compone; `internal/config` lee/valida; `internal/server/httpapi/v1` adapta OpenAPI a casos de uso. | El servidor transversal no debe introducir dependencias en `ticket` ni `draw`; el adaptador v1 queda reservado para operaciones contractuales. |
| `ARCHITECTURE.md`, «Flujo de una solicitud» | El borde transforma errores de dominio a Problem Details. | El manejador central de fallos pertenece al borde HTTP, sin códigos HTTP en futuro negocio. |
| `docs/PRODUCT.md` | No hay endpoints ni reglas de negocio; confianza gateway-servicio e identidad siguen sin decidirse. | No se añaden identidad, confianza de encabezados, tickets, sorteos, MongoDB ni idempotencia. |
| `api/openapi/v1/openapi.yaml`, `paths: {}` | El contrato v1 contiene solamente componentes, incluidos `ProblemDetails` y `Problem`. | No puede registrarse una operación `/v1`; el contrato no autoriza recurso o método alguno. |
| ADR 0002, «Tipos de contenido» y «Errores» | Errores son RFC 9457 con `application/problem+json`; extensiones comunes: `code`, `requestId`, `errors`; 405 incluye `Allow`. | El error central debe construir estos campos de forma consistente y ocultar detalles internos. |
| ADR 0002, «Versionado» | Las operaciones de API usan `/v1`; endpoints operativos ajenos al contrato no lo usan. | Una ruta operativa sería posible en principio fuera de `/v1`, pero AGENTS también exige OpenAPI antes de cualquier endpoint: no hay decisión que resuelva esa tensión para una ruta nueva. |
| `docs/ENGINEERING.md`, «Go» y «Observabilidad» | No hay estado global mutable; se inyectan generadores de ID; no registrar encabezados completos ni datos sensibles. | El generador de request ID debe ser una dependencia pequeña e inyectable; no se acepta ni registra ciegamente un ID entrante. |
| `scripts/harness/check.sh` | Ejecuta `go mod tidy -diff`; cuando haya paquetes, exige `gofmt`, `go vet ./...`, `go test ./...`; bloquea dependencias de dominio hacia `server`, `mongodb`, `config`. | La dependencia Fiber debe dejar `go.mod`/`go.sum` tidies y los nuevos paquetes han de respetar esa dirección. |

El flujo mínimo que sí permite la evidencia es:

```text
señal del proceso -> cmd/api -> internal/config (carga/validación)
                       -> servidor HTTP Fiber (límite, recuperación, request ID)
                       -> respuesta de infraestructura (si está autorizada)
                       -> apagado ordenado
```

No intervienen todavía `internal/ticket`, `internal/draw`, `internal/mongodb` ni
`internal/server/httpapi/v1`, porque no hay operación OpenAPI que adaptar.

## Diseño mínimo coherente propuesto

La partición siguiente no presupone un recurso de producto ni un endpoint
operativo; separa responsabilidades que el ExecPlan ya exige.

| Propietario sugerido | Archivos / paquete | Responsabilidad y contrato interno mínimo |
| --- | --- | --- |
| configuración | `internal/config/config.go`, `internal/config/config_test.go` | `Config` inmutable tras la carga y función de carga/validación. Sus campos deben representar la dirección de escucha, límite de cuerpo y tiempos efectivamente usados por el servidor. La fuente (entorno, flags o ambas), nombres, unidades y valores por defecto no están definidos: deben decidirse antes de fijar una interfaz pública del proceso. |
| borde HTTP | `internal/server/httpapi/server.go`, `problem.go`, `request_id.go` y tests junto a ellos | Construye la aplicación Fiber con configuración ya validada, recuperación previa al manejador de error, generación de ID por solicitud y serialización RFC 9457. Un paquete `httpapi` no versionado tiene responsabilidad concreta de infraestructura; `httpapi/v1` se crea solo al existir una operación OpenAPI. |
| composition root | `cmd/api/main.go` | Carga config, construye el servidor y coordina señal/cancelación/apagado. No declara handlers de negocio ni reglas. |
| dependencias | `go.mod`, `go.sum` | Añade Fiber v3 y su suma solo como consecuencia de una versión oficialmente verificada y compatible con la versión Go declarada. |

Al agregar archivos reales deben eliminarse los `.gitkeep` correspondientes de
`cmd/api` e `internal/config`, conforme a `docs/ENGINEERING.md`. No existe
todavía `internal/server`; crearlo se justifica únicamente si contiene el
adaptador HTTP concreto anterior, no como directorio anticipado.

### Configuración y lifecycle

La evidencia exige validar antes de arrancar y no usar estado global mutable.
Los invariantes que puede imponer la implementación, sin elegir una política de
producto, son: dirección no vacía y sintácticamente utilizable; límite de cuerpo
positivo; duraciones no negativas/positivas según el significado seleccionado;
y coherencia entre tiempos si la biblioteca la necesita. Valores concretos,
variables de entorno, formato de duración, política ante dirección inválida y
timeout de apagado no aparecen en el repositorio; escogerlos es una decisión
operacional que debe registrarse en el ExecPlan (y una ADR si afecta el
despliegue de manera transversal).

`cmd/api` debe capturar señales del proceso, pedir que el servidor deje de
aceptar trabajo, esperar o aplicar el límite de apagado decidido y devolver un
error observable al fallar el arranque/apagado. La forma exacta de conectar ese
ciclo a Fiber v3 debe confirmarse contra su documentación primaria antes de
codificarla; el repositorio no contiene una abstracción o API existente para
ello.

### Error, request ID y límites de confianza

Para un fallo que alcanza el borde, la respuesta propuesta conserva los campos
RFC 9457 `type`, `title`, `status`, y añade los permitidos por ADR 0002:
`code`, `requestId` y, solo para validación estructurada futura, `errors`.
Los detalles internos de un `panic` o error inesperado no se serializan. El
request ID lo genera el servicio por solicitud; no se toma un identificador
arbitrario recibido. Si más adelante se propaga un ID confiable desde gateway,
primero debe decidirse el límite de confianza listado en `docs/PRODUCT.md`.

No hay en la fuente de verdad una cabecera de respuesta para el request ID, un
formato de ID, ni URIs concretas para `type`. Son decisiones pendientes: no se
deben inventar ni convertir en criterios de éxito. La prueba puede observar el
campo `requestId` de Problem Details sin imponer uno de esos valores.

## Endpoints operativos y la prueba 405

Debe omitirse todo endpoint operativo por ahora, salvo que el responsable
resuelva explícitamente esta contradicción documental:

1. El alcance del ExecPlan excluye health/readiness si no son necesarios y
   prohíbe endpoints de negocio; OpenAPI v1 no tiene operaciones.
2. ADR 0002 permite rutas operativas no versionadas que no pertenezcan al
   contrato de API.
3. `AGENTS.md` establece como regla no negociable que OpenAPI se modifica antes
   de implementar o cambiar un endpoint.

Además, con cero rutas registradas el borde solo puede demostrar «ruta no
encontrada»; no existe una ruta cuyo método sea inválido y, por tanto, no hay
caso 405 genuino ni cabecera `Allow` que probar. Registrar una ruta solo para
test introduce superficie HTTP no especificada. Las alternativas que requieren
decisión del líder son:

- Ajustar el criterio de aceptación para diferir 405 hasta la primera operación
  contractualmente definida. Esta es la opción de alcance mínimo.
- Autorizar y documentar una ruta operativa no versionada, su contrato y su
  finalidad, resolviendo primero la prioridad entre AGENTS y ADR 0002.

No corresponde inventar un `/health`, `/ready`, `/ping` o una ruta de prueba.

## Pruebas y criterios observables

Cuando la decisión anterior esté resuelta, las pruebas proporcionales quedan
partidas así:

- `internal/config`: carga válida, cada error de validación de dirección/límite/
  tiempo y ausencia de estado global. Los valores concretos se derivan de la
  política de configuración decidida.
- `internal/server/httpapi`: solicitudes contra la aplicación Fiber real, no
  solo mocks, para 404 Problem Details, `Content-Type:
  application/problem+json`, un `requestId` generado y un panic convertido a
  500 sin texto sensible. Incorporar el caso 405 y `Allow` únicamente tras
  registrar una ruta autorizada.
- límite de cuerpo: una solicitud que lo exceda produce la semántica de error
  acordada y no llega al handler. El status/cuerpo exacto debe comprobarse
  contra la versión de Fiber escogida y normalizarse en el borde si no cumple
  ADR 0002.
- lifecycle: construcción con configuración validada; inicio, solicitud y
  apagado ordenado con mecanismo testeable. Para goroutines, señal o recursos
  compartidos, ejecutar `go test -race ./...` como recomienda
  `docs/VERIFICATION.md`.
- verificación final: `gofmt`, `go vet ./...`, `go test ./...` y `./init.sh`,
  anotando fecha y resultado en el ExecPlan. Tras los primeros archivos Go, el
  harness deja de omitir los checks del toolchain.

## Riesgos y decisiones faltantes

1. **API de Fiber v3 / fasthttp:** el repositorio no fija versión ni expone un
   adaptador previo. Se debe contrastar la configuración real de límites,
   timeout, recuperación, prueba en memoria y shutdown con la documentación
   primaria de la versión seleccionada, en especial la semántica de los
   contextos de Fiber/fasthttp frente a `context.Context` de Go. No debe
   asumirse que un contexto del transporte es seguro de retener o propagar tras
   acabar la request.
2. **Errores del framework:** 404, 405, límite de cuerpo y panic pueden tener
   respuestas predeterminadas distintas de Problem Details. El manejador central
   debe cubrirlos o las pruebas expondrán la discrepancia; no basta confiar en
   defaults de Fiber.
3. **Recuperación:** el orden entre middleware de recuperación y el manejador de
   errores determina si un panic se transforma de forma segura. Debe probarse
   con el borde real.
4. **Confianza de request IDs:** aceptar/ecoar una cabecera entrante permitiría
   correlación falsificada mientras no exista autenticación gateway-servicio.
   Generarlo localmente evita esa afirmación de confianza, pero la política de
   propagación futura sigue abierta.
5. **Configuración y operaciones:** nombres de variables, defaults, timeouts y
   presupuesto de shutdown pueden afectar despliegue. No están establecidos;
   no se deben ocultar dentro de constantes sin registrar la decisión.
6. **405 sin contrato/ruta:** es bloqueante para satisfacer literalmente el
   criterio del ExecPlan sin ampliar alcance.
7. **Compatibilidad de toolchain:** `go 1.26.0` es la única versión declarada;
   la elección de Fiber debe verificarse con esa herramienta y con `go mod tidy
   -diff` del harness, no por memoria.

## Partición segura del trabajo

1. El líder decide y deja constancia en el ExecPlan de: fuente/valores de
   configuración, política de shutdown y resolución de 405/ruta operativa.
2. Un implementador con propiedad exclusiva de `internal/config/**` y
   `go.mod`/`go.sum` introduce la dependencia y configuración, incluido su test;
   no toca `cmd/api` ni servidor.
3. Un segundo implementador, después o con propiedad coordinada sobre
   `internal/server/httpapi/**`, implementa el borde Fiber y pruebas reales. No
   modifica OpenAPI salvo decisión expresa del paso 1.
4. Un tercer alcance exclusivo para `cmd/api/**` conecta configuración,
   lifecycle y señales; puede requerir una prueba de proceso/integración si se
   define cómo controlar el listener.
5. Un reviewer independiente revisa el diff completo, los límites de confianza,
   Problem Details, comportamiento 404/405/panic/límites y evidencia de los
   comandos. El líder actualiza el ExecPlan y ejecuta `./init.sh` final.

Para evitar conflicto de escritura, si un único cambio debe actualizar
`go.mod`/`go.sum`, su propietario lo hace después de que los demás alcances hayan
declarado las importaciones exactas, o integra esos archivos al final.
