# 0004. Organizar el código por vertical slices

- Estado: Aceptada
- Fecha: 2026-08-30
- Responsables: Equipo de cronos

## Contexto

Keno4min tendrá funcionalidades diferenciadas como tiquetes, sorteos,
liquidaciones y jackpots. La estructura debe mantener junto el código de cada
funcionalidad sin crear capas globales ni archivos de servicio demasiado grandes.

## Decisión

El código de negocio se organizará por *vertical slices* dentro de `internal`.
Cada slice agrupará su dominio, casos de uso y adaptadores, manteniendo las
dependencias dirigidas hacia sus contratos.

La estructura vigente y sus reglas se mantendrán en la
[guía de arquitectura](../architecture.md).

## Alternativas consideradas

- Capas globales de Clean Architecture: dispersan una funcionalidad entre varias
  carpetas.
- Un único paquete por funcionalidad: ofrece menos separación entre negocio,
  transporte y persistencia.

## Consecuencias

- El código de una funcionalidad será fácil de localizar, probar y modificar.
- Los slices podrán tener estructuras diferentes según sus necesidades.
- Existirá algo de repetición de carpetas y ensamblaje de dependencias.

## Referencias

- [Product Requirements Document](../product-requirements.md)
- [Guía de arquitectura](../architecture.md)
- [Go Vertical Slice Architecture](https://github.com/sebajax/go-vertical-slice-architecture)
