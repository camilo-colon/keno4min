# ADR 0002: Convenciones de la API HTTP

## Estado

Aceptada.

## Contexto

El servicio expondra una API HTTP interna. Todos sus endpoints necesitan seguir
las mismas reglas de diseno para que el contrato sea predecible, documentable y
compatible con generacion de codigo.

Este ADR define exclusivamente convenciones del contrato HTTP. No define
recursos, operaciones, reglas de negocio, autenticacion, autorizacion,
observabilidad, persistencia ni despliegue.

## Decision

### Estilo de la API

La API seguira un estilo REST pragmatico orientado a recursos.

- Las rutas representaran recursos mediante sustantivos.
- Los recursos se nombraran en plural.
- Las rutas se escribiran en minusculas y utilizaran `kebab-case` para nombres
  compuestos.
- Las rutas no tendran una barra final.
- Los parametros de ruta y consulta utilizaran `lowerCamelCase`.
- Los identificadores se ubicaran despues del recurso que identifican.
- Las acciones que no correspondan naturalmente a CRUD se modelaran como
  comandos explicitos y utilizaran `POST`.
- No se agregara a cada ruta un prefijo que repita el nombre del servicio. Los
  prefijos de enrutamiento externos son responsabilidad del gateway.

El diseno privilegiara una operacion comprensible sobre una representacion CRUD
artificial.

### Semantica de los metodos HTTP

Los metodos conservaran la semantica definida por HTTP:

| Metodo | Uso |
| --- | --- |
| `GET` | Consultar sin producir cambios de estado intencionales |
| `POST` | Crear un recurso o ejecutar un comando no idempotente por naturaleza |
| `PUT` | Reemplazar completamente un recurso en una ubicacion conocida |
| `PATCH` | Aplicar una modificacion parcial con formato documentado |
| `DELETE` | Eliminar un recurso cuando la eliminacion sea la semantica real |

`GET`, `PUT` y `DELETE` respetaran sus propiedades de idempotencia. `PATCH` no se
utilizara como un comando generico para ocultar transiciones de estado.

### Contrato OpenAPI

OpenAPI sera la fuente de verdad del contrato HTTP. Cada version mayor soportada
tendra su propio documento.

El contrato describira como minimo:

- rutas, metodos y `operationId`;
- parametros y encabezados;
- cuerpos de solicitud y respuesta;
- tipos de contenido;
- codigos de estado;
- esquemas y restricciones;
- errores y ejemplos.

Todo endpoint implementado debera existir previamente en OpenAPI. Los tipos HTTP
generados a partir del contrato permaneceran separados de los tipos internos de
la aplicacion. El codigo generado no contendra reglas de negocio y no se editara
manualmente.

### Versionado del contrato HTTP

La version mayor del contrato sera el primer segmento de las rutas:

```text
/v1/...
```

Solo se expondra la version mayor. No se utilizaran versiones minor o patch en
la URL, como `/v1.1` o `/v1.2.3`.

- Los cambios compatibles se incorporaran a la version existente.
- Los cambios incompatibles requeriran una nueva version mayor, por ejemplo
  `/v2`.
- Dos versiones mayores podran coexistir durante un periodo de migracion.
- Una version retirada no se reutilizara para un contrato diferente.
- Los endpoints operativos que no pertenezcan al contrato de la API no usaran el
  prefijo de version.

Agregar una operacion o un campo opcional sera compatible cuando los consumidores
esten obligados a tolerar propiedades desconocidas. Eliminar o renombrar
operaciones o campos, cambiar tipos, agregar requisitos obligatorios o modificar
la semantica existente se considerara incompatible.

Los contratos se organizaran por version:

```text
api/openapi/v1/openapi.yaml
api/openapi/v2/openapi.yaml
```

El campo `openapi` identificara la version de la especificacion utilizada para
interpretar el documento. El campo `info.version` identificara la revision del
documento y no reemplazara la version mayor presente en las rutas.

Cuando una version vaya a retirarse, se marcara como obsoleta en OpenAPI, se
comunicara su reemplazo y se permitira un periodo de migracion. Cuando sea
aplicable, las respuestas utilizaran los encabezados HTTP `Deprecation` y
`Sunset`.

### Tipos de contenido

Las solicitudes y respuestas estructuradas utilizaran JSON:

```http
Content-Type: application/json
```

Los errores utilizaran:

```http
Content-Type: application/problem+json
```

Un endpoint declarara explicitamente en OpenAPI cualquier tipo de contenido
adicional que necesite admitir.

### Representacion JSON

Los nombres de propiedades utilizaran `lowerCamelCase`:

```json
{
  "resourceId": "01K...",
  "createdAt": "2026-08-18T03:15:00Z"
}
```

Se aplicaran las siguientes reglas:

- los identificadores se expondran como strings opacos;
- las fechas y horas utilizaran RFC 3339 y se expresaran en UTC;
- los valores booleanos y numericos conservaran su tipo JSON y no se codificaran
  como strings;
- los valores enumerados utilizaran `snake_case` en minusculas;
- un campo ausente y un campo con valor `null` no se consideraran equivalentes;
- `null` solo se admitira cuando tenga un significado definido en el contrato;
- las propiedades desconocidas no deberan provocar fallos en consumidores, salvo
  que un contrato particular requiera validacion estricta de entrada.

En Go se conservara la convencion de inicialismos, por ejemplo `ResourceID`,
aunque su representacion JSON sea `resourceId`.

### Respuestas exitosas

Las respuestas exitosas devolveran directamente el recurso o resultado. No se
utilizara un envoltorio global con campos como `code`, `message` y `data`, porque
el estado HTTP ya comunica el resultado general de la operacion.

Las colecciones utilizaran un objeto para permitir paginacion y metadatos sin
cambiar posteriormente la forma de la respuesta:

```json
{
  "items": [],
  "nextCursor": null
}
```

Cuando una coleccion necesite paginacion se preferira un cursor opaco. El orden,
el limite permitido y el significado del cursor se documentaran por endpoint.

Los codigos exitosos se seleccionaran por su semantica:

| Estado | Uso |
| --- | --- |
| `200 OK` | Operacion completada con una representacion en la respuesta |
| `201 Created` | Recurso creado; incluira `Location` cuando sea identificable por URI |
| `202 Accepted` | Procesamiento asincrono aceptado pero no finalizado |
| `204 No Content` | Operacion completada sin cuerpo de respuesta |

Una respuesta `204 No Content` no incluira cuerpo.

### Errores

Los errores utilizaran Problem Details conforme a RFC 9457.

```json
{
  "type": "https://errors.example.com/resource-not-found",
  "title": "Resource not found",
  "status": 404,
  "detail": "The requested resource does not exist",
  "instance": "/v1/resources/01K...",
  "code": "resource_not_found",
  "requestId": "req_01K..."
}
```

Los campos estandar `type`, `title`, `status`, `detail` e `instance` conservaran
la semantica definida por RFC 9457. Las extensiones comunes seran:

- `code`: identificador estable y legible por maquinas en `snake_case`;
- `requestId`: identificador de correlacion de la solicitud;
- `errors`: lista opcional de errores de validacion estructurados.

Los consumidores tomaran decisiones a partir de `status`, `type` o `code`, no a
partir de `title` o `detail`. Los errores internos no expondran trazas, consultas,
credenciales ni detalles de infraestructura.

Se aplicara la siguiente semantica general:

| Estado | Uso |
| --- | --- |
| `400 Bad Request` | Sintaxis, parametros o encabezados mal formados |
| `401 Unauthorized` | No se proporcionaron credenciales validas cuando son requeridas |
| `403 Forbidden` | La identidad es conocida pero no tiene permiso |
| `404 Not Found` | El recurso solicitado no existe o no debe revelarse |
| `405 Method Not Allowed` | La ruta existe pero no admite el metodo; incluira `Allow` |
| `409 Conflict` | La solicitud entra en conflicto con el estado actual |
| `415 Unsupported Media Type` | El tipo de contenido de entrada no es admitido |
| `422 Unprocessable Content` | La sintaxis es valida, pero el contenido no puede procesarse |
| `429 Too Many Requests` | Se excedio un limite; incluira `Retry-After` cuando corresponda |
| `500 Internal Server Error` | Error inesperado no atribuible al consumidor |
| `503 Service Unavailable` | El servicio no puede atender temporalmente la solicitud |

### Idempotencia y reintentos

Las operaciones que puedan reintentarse y produzcan efectos no idempotentes
admitiran el encabezado `Idempotency-Key`. El contrato de cada operacion indicara
si es obligatorio.

- La clave se evaluara dentro del contexto de la operacion y del consumidor.
- La misma clave con la misma solicitud devolvera el resultado original.
- La misma clave con una solicitud diferente producira `409 Conflict`.
- La implementacion conservara una huella de la solicitud y la informacion
  necesaria para reproducir la respuesta.
- El periodo de retencion se documentara antes de habilitar la operacion.

La idempotencia HTTP no sustituira los controles atomicos y de concurrencia en la
aplicacion o la persistencia.

## Alternativas consideradas

### Aplicar REST como CRUD estricto

Se descarta porque no todas las operaciones se representan claramente como
creacion, lectura, reemplazo o eliminacion. Los comandos explicitos permiten
conservar la semantica del protocolo sin ocultar la intencion de la operacion.

### Utilizar un envoltorio comun para todas las respuestas

Se descarta porque duplica la semantica de HTTP y agrega estructura sin valor a
las respuestas de un unico recurso.

### Utilizar un formato de error propio

Se descarta para evitar mantener una convencion propietaria cuando RFC 9457 ya
define un formato extensible e interoperable.

### Definir requests y responses manualmente en archivos globales

Se descarta porque concentra contratos no relacionados y favorece el
acoplamiento. OpenAPI definira los esquemas HTTP y el codigo manual se organizara
por operacion.

### Versionar mediante encabezados o tipos de contenido

Se descarta para la version mayor porque hace menos visible el contrato activo y
complica el enrutamiento y la documentacion. La version mayor se mantendra en la
ruta.

## Consecuencias

- Los endpoints compartiran una semantica HTTP consistente.
- Los consumidores podran interpretar respuestas y errores de forma uniforme.
- El contrato podra validarse antes de implementar handlers.
- La generacion de tipos reducira diferencias entre documentacion e
  implementacion.
- Cada operacion debera documentar explicitamente las excepciones a estas reglas.
- OpenAPI, el codigo generado y la implementacion deberan mantenerse
  sincronizados.

## Referencias

- [OpenAPI Specification](https://spec.openapis.org/oas/)
- [RFC 9110: HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html)
- [RFC 9457: Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html)
- [RFC 3339: Date and Time on the Internet](https://www.rfc-editor.org/rfc/rfc3339.html)
- [Idempotency-Key HTTP Header Field](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header)
- [AIP-185: API Versioning](https://google.aip.dev/185)
- [RFC 9745: The Deprecation HTTP Response Header Field](https://www.rfc-editor.org/rfc/rfc9745.html)
- [RFC 8594: The Sunset HTTP Header Field](https://www.rfc-editor.org/rfc/rfc8594.html)
