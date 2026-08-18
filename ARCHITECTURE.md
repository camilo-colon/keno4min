# Arquitectura de Keno4min

## Proposito

Keno4min es un servicio interno responsable de tickets y sorteos. El gateway es
el unico cliente de red previsto y es responsable de autenticar al consumidor.
El contrato HTTP versionado vive en `api/openapi/`.

Este documento es el mapa arquitectonico. Los detalles y la razon de decisiones
estables viven en `docs/adr/`.

## Contexto

```text
consumidor -> gateway -> API HTTP Keno4min -> casos de uso -> MongoDB
                                |
                                +-> observabilidad/configuracion
```

El encabezado exacto de identidad, la topologia de despliegue y el modelo de
datos aun no estan definidos. No deben deducirse del diagrama.

## Paquetes y responsabilidades

| Ruta | Responsabilidad |
| --- | --- |
| `cmd/api` | Composition root, ciclo de vida del proceso y wiring |
| `internal/server/httpapi/v1` | Adaptacion entre HTTP/OpenAPI y casos de uso |
| `internal/ticket` | Entidades, reglas y casos de uso de tickets |
| `internal/draw` | Entidades, reglas y consultas de sorteos |
| `internal/mongodb` | Implementaciones MongoDB de puertos consumidos por negocio |
| `internal/config` | Lectura y validacion de configuracion del proceso |
| `api/openapi/v1` | Contrato HTTP v1; fuente de verdad externa |
| `migrations/mongodb` | Indices y transformaciones versionadas y recuperables |

No se crean subpaquetes hasta que exista una responsabilidad concreta.

## Direccion de dependencias

```text
cmd/api -------------------------------> config
   |                                      |
   +-> server/httpapi/v1 -> ticket/draw <-+ (solo wiring)
   |                           ^
   +-> mongodb ----------------+
```

Reglas:

- `ticket` y `draw` pueden depender de la biblioteca estandar y de tipos propios.
- `server/httpapi/v1` puede depender de los paquetes de negocio, nunca al reves.
- `mongodb` puede depender de los contratos de negocio que implementa, nunca al
  reves.
- `config` no expone variables globales ni se filtra a las reglas de negocio.
- `cmd/api` es el unico lugar que conoce todas las implementaciones concretas.
- Una dependencia directa entre `ticket` y `draw` exige modelar primero el caso
  de uso y registrar la decision si crea acoplamiento estable.

El script `scripts/harness/check.sh` aplica las prohibiciones mas importantes.

## Flujo de una solicitud

1. El transporte valida forma, tipos y metadatos confiables definidos por el
   contrato.
2. Traduce DTO HTTP a una entrada del caso de uso.
3. El negocio aplica invariantes sin conocer HTTP ni MongoDB.
4. Los adaptadores ejecutan I/O mediante interfaces definidas por el consumidor.
5. El transporte traduce el resultado a la respuesta declarada en OpenAPI.

Los errores de dominio no contienen codigos HTTP. La traduccion a Problem
Details sucede en el borde HTTP.

## Contratos y datos

- OpenAPI precede al handler y a cualquier tipo generado.
- Cada cambio de esquema MongoDB incluye una migracion o una explicacion escrita
  de por que no la necesita.
- La atomicidad, los indices unicos y la idempotencia se disenan juntos; una
  comprobacion seguida de una escritura no se considera proteccion suficiente.
- Los formatos persistidos no se reutilizan automaticamente como DTO HTTP.

## Cambios arquitectonicos

Crea una ADR cuando una decision sea costosa de revertir, afecte varios paquetes,
cambie un limite de confianza o introduzca una tecnologia transversal. Un
refactor local y reversible puede documentarse solo en su ExecPlan.
