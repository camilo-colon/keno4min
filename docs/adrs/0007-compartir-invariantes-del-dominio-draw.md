# 0007. Compartir invariantes del dominio draw

- Estado: Reemplazada
- Reemplazada por: [ADR 0008](0008-organizar-por-funcionalidad-y-adaptadores.md)
- Fecha: 2026-09-04
- Responsables: Equipo de cronos

## Contexto

`getbytir` validaba estado, tiempos y bolas dentro de su adaptador DynamoDB,
sin usar la entidad existente. Otra consulta del draw completo, como `getbyid`,
tendría que duplicar esas reglas. La creación ya valida parte de ellas.

## Decisión

Las invariantes del draw completo pertenecen a `draw.Draw.Validate()`.
`draw.New` reutiliza este método. Los slices pueden importar `draw`, que no
depende de transporte, persistencia ni operaciones concretas.

El store de `getbytir` devuelve `draw.Draw`. Comprueba el formato DynamoDB y
la correspondencia del lookup con el canónico. La consulta valida la entidad
y construye su resultado HTTP. Una entidad almacenada inválida es un error
interno, no un error en los parámetros de la petición.

Esta decisión complementa el ADR 0006: mantiene los slices por operación y
concreta una responsabilidad común estable. Las proyecciones parciales no
están obligadas a reconstruir ni validar un draw completo.

## Alternativas consideradas

- Duplicar invariantes por slice: permite que las reglas diverjan.
- Mantenerlas en DynamoDB: acopla reglas de negocio al almacenamiento.
- Crear un repositorio genérico: no es necesario para compartir invariantes.

## Consecuencias

- Creación y consultas del draw completo comparten las reglas y sus pruebas.
- Cada consulta debe validar la entidad recibida antes de utilizarla.
- Los tipos de almacenamiento y respuesta permanecen específicos del slice.

## Referencias

- [ADR 0006](0006-organizar-slices-por-operacion.md)
- [Arquitectura vigente](../architecture.md)
- [Patrones de acceso de draws](../data-model/draw-access-patterns.md)
