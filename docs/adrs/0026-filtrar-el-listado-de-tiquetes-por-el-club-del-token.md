# 0026. Filtrar el listado de tiquetes por el club del token

- Estado: Aceptada
- Fecha: 2026-10-28
- Responsables: Equipo de cronos

## Contexto

El listado por nodo y rango de creación (`AP-TICKET-05`) recibía `nodeId` en
la query, de modo que cualquier cliente autenticado podía consultar los
tiquetes de otro club. La creación ya toma el club del claim `node_id` del
token ([ADR 0025](0025-tomar-club-y-vendedor-de-los-claims-del-token.md)).

## Decisión

El handler del listado toma `Identity.NodeID` (claim `node_id`) mediante
`auth.FromContext` y lo entrega en `ports.ListParams.NodeID`. La query admite
`from`, `to`, `limit` y `cursor`; si incluye `nodeId`, se rechaza con `400`
para que el cliente no asuma que filtra por otro club. Si la identidad no está
disponible, responde `401`. El caso de uso y el repositorio no cambian.

## Alternativas consideradas

- Ignorar `nodeId` en la query: no rompe a los clientes actuales, pero oculta
  que el filtro ya no se aplica.
- Aceptar `nodeId` y exigir que coincida con el token: conserva un parámetro
  sin información y no cubre jerarquías con varios clubes, que siguen fuera de
  esta etapa.

## Consecuencias

- Cada usuario solo lista los tiquetes de su club.
- Los clientes deben dejar de enviar `nodeId`.
- Aplica la condición del
  [ADR 0023](0023-delegar-la-validacion-del-token-al-api-gateway.md): la
  restricción depende de que la API solo sea accesible a través del API Gateway.
- Un listado para roles que supervisan varios clubes requerirá una decisión
  posterior.

## Referencias

- [Modelo de tiquetes](../data-model/ticket-access-patterns.md).
- [Arquitectura](../architecture.md).
