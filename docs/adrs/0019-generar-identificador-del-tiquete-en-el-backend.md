# 0019. Generar el identificador del tiquete en el backend

- Estado: Aceptada
- Fecha: 2026-10-24
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0017](0017-conservar-vendedor-como-created-by.md)

## Contexto

Se acuerda retirar el identificador del tiquete de la entrada del frontend y
asignarlo en el backend. La idempotencia de la venta sigue pendiente de diseño.

## Decisión

`GET /tickets` recibe `userId`, `stv` y `sum`, sin body ni identificador del
tiquete en la query o en la ruta. El caso de uso genera un UUID v7 mediante
`uuid.NewV7()` antes de construir y guardar el tiquete. `CreateParams` deja de
recibir `ID`; el constructor de dominio conserva su parámetro UUID interno.

Se mantienen las demás decisiones del ADR 0017: `CreatedBy` conserva el
username del usuario consultado y `NodeID` su club; el UUID del usuario no se
persiste. Se conservan el contador global, la reserva exclusiva del cupón,
el TIR y las validaciones y aplazamientos del flujo actual.

## Consecuencias

- El frontend deja de generar y enviar el UUID del tiquete.
- Las claves y atributos persistidos mantienen su formato; no se requiere
  migrar datos existentes.
- Una nueva ejecución genera otro UUID. El cambio no resuelve los reintentos
  ni evita ventas duplicadas; el diseño de idempotencia sigue pendiente.
- La integración del handler HTTP y su respuesta siguen pendientes.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Dominio y persistencia del tiquete](../data-model/ticket-access-patterns.md).
