# 0018. Separar la composición Fx del dominio Draw

- Estado: Aceptada
- Fecha: 2026-10-24
- Responsables: Equipo de cronos

## Contexto

El repositorio de sorteos debe devolver entidades `Draw`. Su puerto necesita
importar el dominio, pero `draws/module.go` importaba a su vez los puertos y
adaptadores, lo que produciría un ciclo de dependencias.

## Decisión

Mover la composición Fx a `draws/drawsfx`, siguiendo la organización de
`users/usersfx` y `nodes/nodesfx`. `cmd/api` consume `drawsfx.Module`; el
paquete raíz de sorteos conserva únicamente el dominio.

## Consecuencias

- Los puertos y adaptadores pueden reutilizar `Draw` sin importar Fx.
- `GetLatestDraws` devuelve `[]draw.Draw`; `LatestResult` se conserva como
  contrato de la proyección runtime.
- El registro de proveedores y rutas mantiene el mismo comportamiento.

## Referencias

- [Arquitectura](../architecture.md#dependencias).
