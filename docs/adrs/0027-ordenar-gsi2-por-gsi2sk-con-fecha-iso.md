# 0027. Ordenar GSI2 por GSI2SK y persistir la fecha en ISO con milisegundos

- Estado: Aceptada
- Fecha: 2026-10-28
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0022](0022-indexar-tiquetes-y-persistir-fecha-en-milisegundos.md)

## Contexto

El [ADR 0022](0022-indexar-tiquetes-y-persistir-fecha-en-milisegundos.md)
definió `createdAt` (Number, milisegundos Unix) como sort key de `GSI2`. El
índice desplegado por el repositorio central de infraestructura usa `GSI2SK`
(String). El tiquete no escribía `GSI2SK`, por lo que DynamoDB no lo indexaba,
y la consulta sobre `createdAt` no corresponde al esquema del índice. Sin esa
función de sort key, el formato numérico de `createdAt` ya no aporta ventajas.

## Decisión

Persistir `createdAt` como String con `CreatedAt` en UTC y formato ISO 8601 de
ancho fijo con milisegundos (`2006-01-02T15:04:05.000Z07:00`). El tiquete
escribe el mismo valor en `GSI2SK`, de modo que el orden lexicográfico del
índice coincide con el cronológico. El listado consulta
`GSI2SK BETWEEN from AND to-1ms` con valores String.

La lectura interpreta `createdAt` con `time.RFC3339Nano`, que también acepta
los tiquetes guardados con precisión variable. Las claves de `GSI1` y el resto
del ADR 0022 no cambian.

## Alternativas consideradas

- Conservar `createdAt` como Number y agregar solo `GSI2SK`: menor cambio, pero
  duplica la fecha en dos formatos distintos.
- Recrear `GSI2` con `createdAt` (Number) como sort key: exige cambiar la
  infraestructura compartida y reconstruir el índice.
- `time.RFC3339` (segundos): pierde la precisión de milisegundos que exigen los
  límites del caso de uso.
- `time.RFC3339Nano` para escribir: recorta ceros finales, el ancho varía y el
  orden lexicográfico deja de ser cronológico.

## Consecuencias

- Los tiquetes se indexan en `GSI2` desde su creación y la fecha es legible al
  inspeccionar la tabla.
- `createdAt` y `GSI2SK` repiten el mismo valor y deben derivarse del mismo
  instante.
- Los tiquetes sin `GSI2SK` no aparecen en el listado.

## Referencias

- [Modelo de tiquetes](../data-model/ticket-access-patterns.md).
- [Índices dispersos de DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/bp-indexes-general-sparse-indexes.html).
