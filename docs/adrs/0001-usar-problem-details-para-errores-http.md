# 0001. Usar Problem Details para los errores HTTP

- Estado: Aceptada
- Fecha: 2026-08-28
- Responsables: Equipo de cronos

## Contexto

La API necesita un contrato de errores uniforme, predecible y comprensible para
sus consumidores. Se adopta un estándar para evitar un formato propio y
facilitar la interoperabilidad con clientes y herramientas.

## Decisión

Todas las respuestas HTTP con estado `4xx` o `5xx` usarán Problem Details
conforme a RFC 9457 y el tipo de contenido `application/problem+json`.

`request_id` será una extensión común y tendrá el mismo valor que el encabezado
`X-Request-ID`, para correlacionar la respuesta con los logs. Podrán definirse
otras extensiones, siempre que formen parte del contrato documentado del tipo de
problema.

Los detalles enviados al cliente no expondrán información interna ni sensible.
Los consumidores identificarán los problemas mediante `type` y el estado HTTP,
no interpretando los textos de `title` o `detail`.

## Consecuencias

- Los errores tendrán una representación estándar y extensible.
- Los tipos de problema propios del dominio formarán parte del contrato público
  de la API y deberán permanecer estables.
- El equipo deberá documentar las extensiones y proteger la información incluida
  en cada respuesta.

## Referencias

- [RFC 9457: Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html)
