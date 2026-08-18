# ADR 0003: Fiber v3 como servidor HTTP

## Estado

Aceptada.

## Contexto

Keno4min necesita un borde HTTP que aplique las convenciones de la API, limites
operativos y apagado ordenado sin mezclar el transporte con los casos de uso.
Antes de esta decision el repositorio no tenia servidor ni framework HTTP.

Elegir el servidor es una decision transversal: todos los adaptadores HTTP
futuros dependeran de su enrutamiento, modelo de contexto, manejo de errores,
limites y ciclo de vida. Fiber esta construido sobre `fasthttp`, cuyas
semanticas de memoria y shutdown difieren de `net/http` y deben quedar
explicitas.

Esta ADR no define endpoints, recursos, identidad, autenticacion, reglas de
negocio, persistencia ni confianza en encabezados de proxy. Las operaciones
productivas siguen requiriendo contrato OpenAPI previo.

## Decision

### Framework y alcance

El proceso API usara Fiber v3, fijado a una version exacta en `go.mod`. El
paquete `internal/server/httpapi` es propietario de la aplicacion Fiber, del
manejo transversal de errores y del ciclo de vida del listener. `cmd/api`
conserva la responsabilidad de cargar configuracion, crear el listener, recibir
senales del proceso y ensamblar dependencias.

La adopcion inicial no registra endpoints. Las rutas usadas para verificar
routing, errores, limites o shutdown existen exclusivamente como fixtures de
test. Cada endpoint futuro debe existir primero en OpenAPI y se registrara en el
adaptador de la version correspondiente.

### Contexto y limite del transporte

`fiber.Ctx` y los valores obtenidos de el son validos solo durante el handler;
Fiber/fasthttp reutiliza estructuras y buffers entre solicitudes. Ningun caso de
uso, entidad, repositorio o goroutine que sobreviva al handler puede recibir,
guardar o retener `fiber.Ctx` ni referencias a sus datos.

Los handlers traduciran la solicitud a tipos propios y, cuando una operacion
necesite contexto, pasaran un `context.Context` apropiado. No se asumira que
`fiber.Ctx` ofrece por si mismo cancelacion por desconexion: cualquier garantia
de cancelacion se verificara contra la version fijada antes de depender de ella.

### Errores, limites y confianza

Fiber se configura con limites de cuerpo y timeouts positivos validados antes
del arranque. Recover se instala antes de handlers productivos y un manejador
central traduce errores a Problem Details sin exponer panics, errores internos
ni detalles de infraestructura.

El servicio no confia en proxy headers mientras no exista una lista y un limite
de confianza aprobados. Los identificadores de solicitud usados por el borde se
generan localmente; no se propaga ni publica una cabecera de correlacion hasta
que el contrato entre gateway y servicio la defina.

### Ownership del shutdown

`internal/server/httpapi.Server` posee explicitamente el lifecycle una vez que
recibe un listener. Sincroniza que Fiber haya completado el arranque antes de
observar cancelacion, inicia el listener sin delegar el shutdown a una goroutine
interna del framework y, al cancelar el contexto, llama y espera el shutdown
con el presupuesto configurado. Tambien espera el resultado del serve loop y
combina cualquier error de shutdown y servicio.

El proceso no puede declarar exito ni terminar mientras el drenaje siga en
curso. Si una solicitud supera el presupuesto, el error de timeout se propaga
al composition root. Un contexto cancelado antes del arranque cierra el
listener sin iniciar Fiber.

### Versionado y actualizacion

Fiber se fija dentro de la version mayor v3. Las actualizaciones no se adoptan
automaticamente: cada cambio de version requiere revisar notas de release y
compatibilidad de Fiber/fasthttp, ejecutar pruebas del borde y lifecycle con
`-race`, y confirmar limites, middleware, errores, contextos y shutdown. Un
cambio de version mayor o de framework requiere una ADR que reemplace esta.

## Alternativas consideradas

### Biblioteca estandar `net/http`

Reduciria dependencias y ofrece contextos y shutdown familiares, pero exige
componer por separado routing, recuperacion y otras piezas del borde. Se
descarta para esta etapa porque Fiber v3 proporciona el servidor y routing
coherentes que se desea estandarizar, con sus diferencias cubiertas por este
limite y pruebas.

### Otro router o framework sobre `net/http`

Routers como `chi` o frameworks como Echo evitarian algunas particularidades de
fasthttp. No existe una necesidad del producto que haga superior otra opcion en
esta etapa, y mantener varias pilas HTTP aumentaria la superficie transversal.
Se descartan para conservar una sola implementacion del borde.

### Delegar el apagado al contexto de Fiber

Fiber permite entregar un `GracefulContext` al listener. Se descarta porque la
goroutine interna puede cerrar el listener y permitir que el serve loop retorne
antes de que el drenaje termine, y sus errores de shutdown no llegan al caller.
El servidor del repositorio debe poseer y esperar ambas operaciones.

### Postergar la eleccion hasta el primer endpoint

Evitaria una dependencia temprana, pero impediria verificar desde ahora la
configuracion, Problem Details, limites y lifecycle requeridos para cualquier
flujo vertical. Se descarta porque esa infraestructura ya es una responsabilidad
concreta y probada, aun sin publicar rutas.

## Consecuencias

- Los adaptadores HTTP futuros comparten una aplicacion Fiber v3 y un manejo
  uniforme de errores, limites y shutdown.
- Fiber y fasthttp quedan confinados al borde HTTP; el negocio permanece
  independiente del framework.
- El equipo debe tratar el lifetime de `fiber.Ctx` y sus buffers como una
  restriccion de seguridad y correccion, especialmente ante goroutines.
- El lifecycle requiere pruebas deterministas de drenaje limpio, timeout y
  cancelacion previa al arranque, ademas de pruebas del borde con `app.Test` o
  listener en memoria segun la semantica verificada.
- Actualizar Fiber puede requerir adaptar integraciones aun dentro de v3 y no se
  considera un cambio mecanico sin verificacion.
- No se amplia la superficie HTTP: esta decision no autoriza endpoints
  productivos u operativos.

## Referencias

- [Documentacion oficial de Fiber](https://docs.gofiber.io/)
- [Fiber: Go Context](https://docs.gofiber.io/guide/go-context/)
- [Fiber: Error handling](https://docs.gofiber.io/guide/error-handling/)
- [Fiber: API de aplicacion y shutdown](https://docs.gofiber.io/api/fiber/)
