# 0025. Tomar el club y el vendedor de los claims del token

- Estado: Aceptada
- Fecha: 2026-10-28
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0024](0024-tomar-el-vendedor-del-token-al-crear-tiquetes.md)

## Contexto

El ADR 0024 tomaba el UUID del vendedor del token, pero seguía leyendo la
réplica local de usuarios para obtener su nodo y username. Esa réplica solo se
alimenta con `cronos.core.user.created`: un cambio de nodo en cronos-core no
llega a Keno4min, y un usuario recién creado se rechaza con `404` hasta que el
evento se procese. El token, emitido por cronos-core, ya incluye `node_id`,
que corresponde al club del vendedor, y `unique_name`.

## Decisión

El handler de creación copia `Identity.NodeID` (claim `node_id`) y
`Identity.Username` (claim `unique_name`) a `CreateParams.NodeID` y
`CreateParams.CreatedBy`. `tickets/usecase.Create` deja de depender de
`users/ports` y ya no consulta al usuario ni devuelve `ErrUserNotFound`.
`cmd/api` deja de cargar `usersfx.RepositoryModule`.

Se mantienen las demás decisiones del ADR 0024: la query solo admite `stv` y
`sum`, el backend genera el UUID v7 y el UUID del usuario no se persiste.

## Alternativas consideradas

- Tomar el nodo del token y conservar la lectura del usuario para comprobar
  su existencia: mezcla dos fuentes y mantiene la lectura y el `404` mientras la
  réplica esté desactualizada.
- Mantener la lectura de la réplica: conserva el club anterior cuando el
  usuario cambia de nodo.

## Consecuencias

- El club y el vendedor reflejan cronos-core al emitirse el token; un cambio
  posterior se aplica cuando el cliente obtiene un token nuevo.
- Se elimina una lectura fuerte por venta y la respuesta `404` de la creación.
- La confianza en el API Gateway se amplía: un token falsificado fuera del
  gateway permitiría vender a nombre de cualquier club, no solo de usuarios
  replicados. Aplica la condición del
  [ADR 0023](0023-delegar-la-validacion-del-token-al-api-gateway.md).
- La réplica de usuarios ya no se consulta en la API; su permanencia se
  evaluará por separado.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Arquitectura](../architecture.md).
- [Modelo de tiquetes](../data-model/ticket-access-patterns.md).
