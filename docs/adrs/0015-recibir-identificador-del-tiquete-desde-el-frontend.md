# 0015. Recibir el identificador del tiquete desde el frontend

- Estado: Reemplazada
- Fecha: 2026-10-20
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0014](0014-asignar-cupones-con-contador-y-reserva.md)
- Reemplazada por: [ADR 0016](0016-recibir-usuario-y-conservar-club-en-el-tiquete.md)

## Contexto

El contrato anterior asignaba el UUID del tiquete en el servidor. Se acuerda
que el frontend genere y envíe ese identificador. El diseño de idempotencia de
la venta se aplaza.

## Decisión

Recibir el UUID v7 del tiquete como parámetro obligatorio `id` en la query de
`GET /tickets`, junto con `stv` y `sum`, sin body ni ID como segmento de ruta.
El servidor valida y conserva ese UUID en lugar de generar otro.

Se mantienen las demás decisiones del ADR 0014: cupón de 16 dígitos asignado
por un contador global atómico en una única región de escritura, saltos
permitidos y reserva permanente guardada junto al tiquete en una transacción
con condiciones de no existencia. El servidor sigue asignando el cupón.

Recibir el UUID no define todavía su uso como clave de idempotencia ni el
comportamiento ante solicitudes repetidas. Esas reglas se estudiarán después.

## Alternativas consideradas

- Generar el UUID en el servidor: se reemplaza por la generación en el frontend
  solicitada para el contrato actual.
- Añadir una clave independiente de operación o usar ya el UUID para resolver
  reintentos: se aplaza el diseño de idempotencia.

## Consecuencias

- El frontend debe generar un UUID v7 y enviarlo al crear el tiquete.
- La condición de no existencia impide sobrescribir un tiquete con el mismo
  UUID; la respuesta ante ese caso sigue pendiente.
- La reserva sigue impidiendo que dos tiquetes aceptados compartan cupón.
- El cambio afecta al diseño documentado; la creación todavía no está
  implementada.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Modelo de acceso de tiquetes](../data-model/ticket-access-patterns.md).
