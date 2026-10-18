# 0006. Organizar slices por operación

- Estado: Reemplazada
- Reemplazada por: [ADR 0008](0008-organizar-por-funcionalidad-y-adaptadores.md)
- Fecha: 2026-09-04
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0004](0004-organizar-el-codigo-por-vertical-slices.md)

## Contexto

Las consultas de sorteos tienen tres patrones independientes: búsqueda por TIR,
histórico y estado de presentación. Se necesita modificar cada operación en un
solo lugar, sin dispersarla entre paquetes de handlers, servicios y repositorios.
Las tareas del workflow se encargan de las escrituras; estos endpoints leen el
estado que producen.

## Decisión

Cada operación se implementa como un paquete Go dentro de su agrupación
funcional: `internal/draw/getbytir`, por ejemplo. El paquete contiene su handler
HTTP, consulta, adaptador DynamoDB y pruebas. Los paquetes se crean cuando se
implementa la operación correspondiente.

El límite de dependencias es el slice completo. Se acepta que un slice importe
Fiber y el SDK AWS; la separación de responsabilidades entre sus archivos es
interna. Los slices no se importan entre sí. Cada uno define sus tipos de
lectura e interfaces pequeñas cuando las necesita.

`cmd/api` construye las dependencias y registra los handlers. `platform` conserva
los clientes técnicos y la configuración transversal del servidor; no alberga
consultas de negocio ni repositorios genéricos.

## Alternativas consideradas

- Capas técnicas por funcionalidad: conservan aislamiento por paquete, pero
  dispersan los cambios de una operación entre varios directorios.
- Adaptadores separados por cada operación: proporcionan aislamiento adicional
  a costa de multiplicar paquetes y contratos para consultas pequeñas.

## Consecuencias

- Una operación puede desarrollarse y probarse dentro de su propio directorio.
- Los cambios de Fiber o del formato DynamoDB pueden afectar a varios slices.
- Los límites entre transporte, consulta y persistencia dentro de un slice se
  revisan como convención, no los impone el compilador.
- Se acepta duplicación pequeña de tipos específicos; se comparte código solo
  cuando existe una responsabilidad común y estable.

## Referencias

- [Arquitectura vigente](../architecture.md)
- [Vertical Slice Architecture](https://www.jimmybogard.com/vertical-slice-architecture/)
- [Organizing a Go module](https://go.dev/doc/modules/layout)
