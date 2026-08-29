# 0002. Establecer convenciones para las respuestas HTTP

- Estado: Aceptada
- Fecha: 2026-08-28
- Responsables: Equipo de cronos

## Contexto

La API necesita respuestas consistentes y fáciles de consumir. Usar la semántica
de HTTP evita contratos redundantes y permite que clientes e infraestructura
interpreten cada resultado correctamente.

## Decisión

Las respuestas exitosas que tengan cuerpo usarán `application/json`. Los
recursos y resultados se devolverán directamente, sin sobres genéricos como
`data`, `success` o `message`.

Las colecciones usarán un objeto con `items` y, cuando exista otra página, un
`next_cursor` opaco. `items` siempre será un arreglo; una colección sin
resultados responderá `200 OK` con `items: []`.

Los códigos y cuerpos respetarán la semántica HTTP:

- `200 OK` para operaciones completadas que devuelven una representación.
- `201 Created` para recursos creados, incluyendo su representación y el
  encabezado `Location`.
- `202 Accepted` para procesamiento asíncrono, indicando cómo consultar su
  estado.
- `204 No Content` para operaciones completadas que no devuelven cuerpo.

Las propiedades JSON usarán `snake_case` y los instantes de tiempo usarán RFC
3339 en UTC. Cada respuesta se documentará en OpenAPI. Las respuestas de error
seguirán [ADR 0001](0001-usar-problem-details-para-errores-http.md).

## Consecuencias

- Los clientes podrán interpretar las respuestas mediante HTTP y contratos JSON
  estables.
- Las colecciones podrán paginarse y evolucionar sin cambiar su forma básica.
- Las excepciones a estas convenciones deberán justificarse y documentarse como
  parte del contrato público de la API.

## Referencias

- [RFC 9110: HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html)
- [Google AIP-132: Standard methods — List](https://google.aip.dev/132)
- [Google AIP-133: Standard methods — Create](https://google.aip.dev/133)
- [Google AIP-158: Pagination](https://google.aip.dev/158)
- [Microsoft REST API Guidelines](https://github.com/microsoft/api-guidelines)
