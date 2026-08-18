# Base de conocimiento

Este es el unico indice de documentacion del repositorio. Cada fuente responde
una pregunta distinta; sigue los enlaces en lugar de repetir sus reglas.

## Ruta corta

1. Empieza en [`../AGENTS.md`](../AGENTS.md) para conocer las reglas y localizar
   la fuente relevante.
2. Lee [`PRODUCT.md`](PRODUCT.md) para negocio y preguntas abiertas, o
   [`../ARCHITECTURE.md`](../ARCHITECTURE.md) para limites del sistema.
3. Antes de editar codigo consulta [`ENGINEERING.md`](ENGINEERING.md); antes de
   declarar terminado sigue [`VERIFICATION.md`](VERIFICATION.md) y
   [`../CHECKPOINTS.md`](../CHECKPOINTS.md).

## Donde registrar conocimiento

| Pregunta | Fuente |
| --- | --- |
| Que hace el producto y que falta decidir | [`PRODUCT.md`](PRODUCT.md) |
| Cuales son los limites y dependencias | [`../ARCHITECTURE.md`](../ARCHITECTURE.md) |
| Como se implementa en este repositorio | [`ENGINEERING.md`](ENGINEERING.md) |
| Como se demuestra que funciona | [`VERIFICATION.md`](VERIFICATION.md) |
| Como se coordina trabajo delegado | [`AGENT_WORKFLOWS.md`](AGENT_WORKFLOWS.md) |
| Que significa estar listo | [`../CHECKPOINTS.md`](../CHECKPOINTS.md) |
| Por que se tomo una decision duradera | `docs/adr/` |
| Que se esta haciendo en un trabajo amplio | `docs/exec-plans/active/` |
| Que produjo un subagente | `docs/agent-reports/<task-id>/` |

## Decisiones aceptadas

- [`0001-package-organization.md`](adr/0001-package-organization.md): organizacion
  inicial de paquetes.
- [`0002-http-api-conventions.md`](adr/0002-http-api-conventions.md): convenciones
  de la API HTTP.

Una ADR aceptada no se reescribe para cambiar la decision. Crea otra ADR que la
reemplace y enlaza ambas.

## Trabajo de varias etapas

Crea un plan desde [`exec-plans/TEMPLATE.md`](exec-plans/TEMPLATE.md) dentro de
`docs/exec-plans/active/`. Actualizalo durante el trabajo y muevelo a
`docs/exec-plans/completed/` al cerrar, sin borrar su historia. Un cambio pequeno
puede mantener un plan breve en la conversacion.

Los informes de agentes se organizan por tarea en
`docs/agent-reports/<task-id>/`; su formato, propiedad y veredictos estan en
[`AGENT_WORKFLOWS.md`](AGENT_WORKFLOWS.md).

## Cuando crear otro documento

Agrega una fuente tematica solo si responde una pregunta nueva, contiene reglas
concretas que se consultan por separado y tiene un evento claro que obliga a
actualizarla. Si no cumple las tres condiciones, amplia una fuente existente o
registra el trabajo futuro en un ExecPlan o issue.
