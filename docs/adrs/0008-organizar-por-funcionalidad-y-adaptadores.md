# 0008. Organizar por funcionalidad y adaptadores

- Estado: Aceptada
- Fecha: 2026-09-04
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0006](0006-organizar-slices-por-operacion.md) y
  [ADR 0007](0007-compartir-invariantes-del-dominio-draw.md)

## Contexto

El paquete por operación dispersa tipos y validaciones al consultar la misma
entidad por diferentes claves. Se busca hacer explícita la reutilización del
dominio y del acceso al draw canónico, manteniendo las reglas fuera de DynamoDB.

## Decisión

Organizar por funcionalidad. `draw` contiene la entidad, sus invariantes, los
casos de uso y sus contratos de lectura. `draw/http` y `draw/dynamodb` son
adaptadores que dependen de `draw`; el dominio y los casos de uso no importan
Fiber ni AWS. Los adaptadores no se importan entre sí.

Cada caso de uso define el contrato pequeño que necesita. El adaptador de
persistencia puede implementar varios contratos y compartir decodificación y
accesos. No se crea un repositorio CRUD genérico. `cmd/api` conecta las piezas.

Se sustituye la implementación por TIR por `draw.GetByID`, expuesta en
`GET /api/v1/draws/{id}` mediante una lectura fuerte. La persistencia devuelve
`draw.Draw`; el caso de uso valida sus invariantes e identidad solicitada; HTTP
construye la respuesta. Las invariantes siguen centralizadas en `Draw.Validate()`,
como establecía el ADR 0007, y `draw.New` también las utiliza.

## Alternativas consideradas

- Mantener paquetes por operación: conserva proximidad de cambios, pero añade
  límites que dificultan compartir el acceso a la misma entidad en este proyecto.
- Usar capas globales: separa responsabilidades, pero dispersa cada funcionalidad
  entre directorios comunes a todo el sistema.

## Consecuencias

- Cambiar un endpoint puede requerir modificar varios paquetes del módulo.
- Las reglas, los contratos y los detalles de transporte y persistencia tienen
  límites de dependencia explícitos.
- El endpoint por TIR deja de estar disponible. Su patrón de acceso queda como
  diseño previsto; no se elimina el modelo de datos de las futuras operaciones.
- Las proyecciones parciales pueden mantener tipos propios sin reconstruir un
  draw completo.

## Referencias

- [Arquitectura vigente](../architecture.md)
- [Patrones de acceso de draws](../data-model/draw-access-patterns.md)
