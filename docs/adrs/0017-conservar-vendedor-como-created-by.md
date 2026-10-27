# 0017. Conservar el vendedor como CreatedBy

- Estado: Reemplazada
- Fecha: 2026-10-21
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0016](0016-recibir-usuario-y-conservar-club-en-el-tiquete.md)
- Reemplazada por: [ADR 0019](0019-generar-identificador-del-tiquete-en-el-backend.md)

## Contexto

El listado debe mostrar el nombre del vendedor conservado al crear la venta.
Se acuerda mantener un único campo `CreatedBy` y retirar `UserID` y `Username`
del tiquete.

## Decisión

Guardar `CreatedBy string`, copiado de `user.Username` al crear el tiquete.
El atributo DynamoDB será `createdBy` y se incluirá en la futura proyección
del listado. Es el nombre histórico del vendedor, no su UUID.

La entrada sigue siendo `GET /tickets` con `id`, `userId`, `stv` y `sum`.
`userId` permite consultar al usuario existente para obtener su username y
`NodeId`, pero su UUID no se almacena en el tiquete. El frontend no suministra
`CreatedBy` directamente. `NodeID` sigue conservando el club de la venta.

Se mantienen la identidad UUID v7 del tiquete enviada por el frontend, el
contador y reserva exclusivos del cupón, y las validaciones y aplazamientos
del flujo actual. El modelo vigente incluye el TIR copiado del draw y omite
`Currency` y `RulesVersion`, según el PRD.

## Alternativas consideradas

- Conservar UUID y username del vendedor: se sustituye por el único campo
  `CreatedBy` solicitado para el modelo actual.
- Consultar el nombre por cada fila del listado: requiere lecturas adicionales
  y mostraría los cambios del usuario en lugar del nombre de la venta.

## Consecuencias

- El listado puede mostrar el vendedor desde el propio tiquete.
- Cambiar el username no modifica `CreatedBy` en ventas anteriores.
- El tiquete deja de conservar la identidad UUID del vendedor. La correlación
  para consultas y pagos debe definirse después, sin suponer usernames únicos
  o inmutables.
- Este cambio actualiza el diseño documentado, sin implementar código.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Dominio y persistencia del tiquete](../data-model/ticket-access-patterns.md).
