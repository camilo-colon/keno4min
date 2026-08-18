# AGENTS.md — mapa para agentes

Este archivo es el punto de entrada, no un manual completo. Usa divulgacion
progresiva: lee primero este mapa y abre solo la fuente de verdad relevante para
la tarea.

## Antes de editar

1. Ejecuta `git status --short` y conserva cambios que no te pertenezcan.
2. Ejecuta `./init.sh`. Si falla, separa un fallo preexistente de uno causado por
   tu cambio y deja evidencia.
3. Lee `ARCHITECTURE.md`, `docs/PRODUCT.md` y las ADR relacionadas con la tarea.
4. Define criterios de aceptacion observables antes de implementar.
5. Si el trabajo es amplio, crea un ExecPlan desde
   `docs/exec-plans/TEMPLATE.md` en `docs/exec-plans/active/`.

## Mapa del conocimiento

| Fuente | Contenido | Cuando leerla |
| --- | --- | --- |
| `ARCHITECTURE.md` | Limites, dependencias y flujo del sistema | Todo cambio estructural |
| `docs/PRODUCT.md` | Hechos del producto y decisiones pendientes | Reglas de negocio o API |
| `docs/ENGINEERING.md` | Convenciones de Go, API, persistencia y tests | Antes de escribir codigo |
| `docs/VERIFICATION.md` | Comandos y evidencia requeridos | Antes de declarar terminado |
| `docs/AGENT_WORKFLOWS.md` | Coordinacion, propiedad e informes | Al delegar trabajo |
| `docs/adr/` | Decisiones aceptadas e historicas | Cuando una decision ya existe |
| `docs/exec-plans/` | Planes activos y completados | Trabajo de varias etapas |
| `.codex/agents/` | Lider y agentes Codex especializados | Al delegar trabajo |
| `docs/agent-reports/` | Evidencia persistente de cada subagente | Durante trabajo delegado |
| `CHECKPOINTS.md` | Criterios objetivos de revision | Auto-revision y revision final |
| `api/openapi/` | Fuente de verdad del contrato HTTP | Cualquier endpoint o DTO HTTP |

`docs/README.md` es el indice completo. Si dos documentos se contradicen,
prevalecen, en este orden: contrato o test ejecutable, ADR aceptada,
`ARCHITECTURE.md`, documento tematico y plan de ejecucion. Corrige la fuente
obsoleta en el mismo cambio.

## Reglas no negociables

- OpenAPI se modifica antes de implementar o cambiar un endpoint.
- `internal/ticket` y `internal/draw` no importan HTTP, MongoDB ni configuracion.
- Las interfaces viven junto al codigo que las consume.
- No crees paquetes genericos como `common`, `utils`, `models`, `interfaces` o
  `helpers`; nombra cada paquete por su responsabilidad real.
- `cmd/api` ensambla dependencias; no contiene reglas de negocio.
- El codigo generado no contiene negocio y nunca se edita a mano.
- Todo comportamiento nuevo o corregido incluye pruebas proporcionales al riesgo.
- No inventes encabezados de identidad, estados de ticket, reglas del sorteo,
  esquemas MongoDB ni garantias de idempotencia: documenta la decision faltante.
- No registres secretos, tickets completos ni datos sensibles.
- No mezcles refactors no relacionados con la tarea actual.

## Flujo de trabajo

Para cambios pequenos, conserva un plan breve en la conversacion. Para cambios
de varias etapas, con riesgo o que continuaran en otra sesion, usa un ExecPlan y
actualiza `Progress`, `Decisions` y `Validation` durante el trabajo, no solo al
final.

El chat principal es el punto de entrada. Para implementar, corregir o
refactorizar, delega la tarea completa a `leader` y espera su resultado. El
lider nunca edita codigo: coordina `explorer`, `implementer` y `reviewer` segun
`docs/AGENT_WORKFLOWS.md`. El ExecPlan es el plan vivo y los informes bajo
`docs/agent-reports/` son la evidencia. No hagas que dos agentes editen los
mismos archivos o escriban el mismo informe a la vez.

Orden recomendado:

1. Reproducir o establecer la linea base.
2. Escribir o actualizar contrato, plan y criterios de aceptacion.
3. Implementar el cambio minimo coherente.
4. Ejecutar tests dirigidos durante el desarrollo.
5. Revisar el diff contra `CHECKPOINTS.md`.
6. Ejecutar `./init.sh` como verificacion final.
7. Mover el ExecPlan terminado a `docs/exec-plans/completed/` y registrar el
   trabajo restante en un ExecPlan o issue concreto.

## Si falta informacion

Busca primero en el repositorio. Si una eleccion cambia reglas de negocio,
seguridad, compatibilidad publica o persistencia irreversible, registra la duda
y pide una decision humana. Para decisiones locales y reversibles, explicita el
supuesto y continua.

## Estado final

Un cambio esta terminado cuando sus criterios son observables, los checks pasan,
la documentacion refleja el comportamiento real, el diff no contiene artefactos
temporales y las limitaciones restantes estan registradas como deuda o trabajo
futuro concreto.
