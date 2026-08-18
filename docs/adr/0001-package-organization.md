# ADR 0001: Organizacion inicial de paquetes

## Estado

Aceptada.

## Contexto

Keno4min se desplegara como un servicio interno y comenzara con dos capacidades
de negocio: tickets y sorteos. El gateway sera responsable de la autenticacion y
sera el unico punto autorizado para acceder al servicio.

## Decision

- Los ejecutables se ubicaran en `cmd`.
- Todo el codigo del servicio sera privado y se ubicara en `internal`.
- El negocio se organizara en los paquetes `ticket` y `draw`.
- HTTP, MongoDB y configuracion permaneceran separados del negocio.
- No se crearan paquetes genericos como `common`, `utils`, `models` o `interfaces`.
- Las interfaces se declararan cerca del codigo que las consume.
- Los tests se ubicaran junto al paquete probado.
- No se agregaran paquetes o directorios especulativos.

## Consecuencias

La estructura comenzara pequena y crecera a partir de responsabilidades reales.
Los paquetes de negocio no dependeran del transporte HTTP ni de MongoDB.
