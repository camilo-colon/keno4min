# Flujos de trabajo con agentes

## Fuentes de verdad

El chat principal recibe la solicitud y delega la coordinacion a `leader`. Las
responsabilidades e instrucciones completas de cada rol viven en:

- [`.codex/agents/leader.toml`](../.codex/agents/leader.toml)
- [`.codex/agents/explorer.toml`](../.codex/agents/explorer.toml)
- [`.codex/agents/implementer.toml`](../.codex/agents/implementer.toml)
- [`.codex/agents/reviewer.toml`](../.codex/agents/reviewer.toml)

Este documento define solo el protocolo compartido, la propiedad de archivos y
la evidencia en disco. El ExecPlan es el plan vivo; el chat transporta
referencias, no sustituye esos artefactos.

## Protocolo

1. `leader` establece la linea base, define criterios observables e identifica o
   crea un ExecPlan para trabajo amplio.
2. Si falta evidencia, delega preguntas independientes a uno o mas `explorer`.
3. Despues de leer los informes, asigna a `implementer` un alcance cerrado y una
   ruta de informe unica.
4. `reviewer` inspecciona el estado real del repositorio despues de cada
   implementacion y emite un veredicto independiente.
5. Ante `CHANGES_REQUESTED`, `leader` delega la correccion y solicita otra
   revision. Solo `APPROVED` permite continuar al cierre.
6. `leader` ejecuta la verificacion final, actualiza y cierra el ExecPlan y deja
   un resumen en disco.

Una tarea trivial puede omitir exploracion y ExecPlan, pero no la separacion
entre implementacion y revision. La escritura paralela solo se permite cuando
los conjuntos de archivos y las rutas de informe son disjuntos.

## Propiedad

Antes de delegar, `leader` asigna por escrito el alcance y una ruta exclusiva
bajo `docs/agent-reports/<task-id>/`.

| Rol | Puede escribir |
| --- | --- |
| `leader` | ExecPlan e informe de cierre |
| `explorer` | Solo `exploration-<tema>.md` asignado |
| `implementer` | Archivos del alcance y `implementation-<alcance>.md` |
| `reviewer` | Solo `review-<alcance>.md` asignado |

Ningun agente modifica el informe de otro ni integra archivos fuera de su
propiedad. Los permisos activos del chat pueden ser mas amplios que el
`sandbox_mode` del TOML; el alcance escrito sigue siendo obligatorio.

## Informes y respuestas

Cada informe destila alcance, archivos relevantes, decisiones, comandos y
resultados, omisiones, bloqueos y riesgo residual. No incluye secretos, datos
sensibles, diffs completos ni logs sin procesar.

La respuesta al agente padre es una sola referencia:

```text
done -> docs/agent-reports/<task-id>/<informe>.md
blocked -> docs/agent-reports/<task-id>/<informe>.md
APPROVED -> docs/agent-reports/<task-id>/review-<alcance>.md
CHANGES_REQUESTED -> docs/agent-reports/<task-id>/review-<alcance>.md
```

`APPROVED` significa que los criterios son observables, la evidencia es
suficiente y no quedan hallazgos bloqueantes. `CHANGES_REQUESTED` enumera cada
hallazgo con impacto, ubicacion y correccion verificable. Un resultado que solo
existe en el chat no se considera terminado.

## Escalacion humana

Escala cuando falte una decision que cambie reglas de negocio, limites de
confianza, compatibilidad publica, persistencia irreversible o costo operacional
significativo. Registra contexto y opciones en el ExecPlan; no escondas la
decision dentro de una implementacion tecnica.
