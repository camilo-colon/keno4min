# Agentes personalizados nativos de Codex

Estado: completed
Creado: 2026-08-18
Responsable: agente principal

## Proposito

Hacer visible y ejecutable el flujo multiagente del repositorio en Codex mediante
agentes de proyecto reales, en lugar de limitarlo a una descripcion generica.

## Contexto

El harness ya describe roles de lider, implementador y revisor, pero no contiene
archivos que Codex descubra como agentes personalizados. Codex usa
`.codex/agents/*.toml`; `.claude/agents/` pertenece a Claude Code y no se debe
copiar a este proyecto.

## Criterios de aceptacion

- [x] Existe `.codex/config.toml` con multiagente habilitado y concurrencia
  acotada.
- [x] Codex puede descubrir `leader`, `explorer`, `implementer` y `reviewer`.
- [x] El chat principal delega la orquestacion al agente `leader` explicito.
- [x] Los agentes tienen permisos e instrucciones proporcionales a su rol.
- [x] `./init.sh` valida la presencia y el esquema minimo de los agentes.
- [x] La documentacion explica como pedir a Codex que los use y donde verlos.
- [x] Cada subagente registra evidencia en disco y responde solo con su ruta.

## Alcance y fuera de alcance

Incluye configuracion de proyecto, cuatro agentes y su integracion documental. No
modifica la configuracion personal en `~/.codex`, no fija credenciales y no
introduce configuracion de Claude Code.

## Plan

1. Crear `.codex/config.toml` y cuatro TOML bajo `.codex/agents/`.
2. Actualizar el mapa y el protocolo de orquestacion.
3. Extender el check estructural del harness.
4. Validar TOML, configuracion Codex y `./init.sh`.

## Progress

- [2026-08-18] Confirmada en OpenAI Docs la ruta y el esquema de custom agents.
- [2026-08-18] Creada la primera version de agentes y la configuracion
  multiagente de proyecto.
- [2026-08-18] Revision independiente confirmo esquema y roles; se corrigieron
  nombres de archivo y documentacion sobre overrides del sandbox.
- [2026-08-18] Los agentes se renombraron como capacidades genericas y se
  retiro de sus instrucciones todo acoplamiento al producto o a su tecnologia.
- [2026-08-18] Agregado `leader`, renombrado el explorador como `explorer` e
  incorporados informes persistentes bajo `docs/agent-reports/`.

## Surprises & discoveries

- La instalacion local reporta `multi_agent` como feature estable y habilitada.
- El thread del explorador fue visible durante la validacion, pero una sesion
  iniciada antes de crear los TOML no expone si recargo el agente personalizado.

## Decision log

- [2026-08-18] Decision inicial, luego reemplazada: el chat principal seria el
  lider y los custom agents serian especialistas delegables.
- [2026-08-18] Las definiciones de agente seran agnosticas al proyecto. El
  contexto concreto vendra de `AGENTS.md` y de las fuentes de verdad del repo.
- [2026-08-18] Decision vigente: el chat principal es el punto de entrada y
  delega la coordinacion a `leader`; este nunca implementa codigo. Los
  subagentes comunican resultados mediante informes en disco.

## Validation

| Comando o prueba | Resultado | Fecha |
| --- | --- | --- |
| `./init.sh` | Verde | 2026-08-18 |
| parseo de todos los TOML con `tomllib` | Valido | 2026-08-18 |
| nombres de agente | `leader`, `explorer`, `implementer`, `reviewer` | 2026-08-18 |
| `codex --strict-config doctor` | `config.load: ok`; multi_agent habilitado | 2026-08-18 |
| revision con un subagente explorador | Esquema y roles coherentes; cautelas corregidas | 2026-08-18 |
| sintaxis Bash, enlaces y whitespace | Sin hallazgos | 2026-08-18 |

## Recuperacion

Los cambios son archivos de configuracion versionados. Pueden revertirse por
archivo y no modifican la configuracion personal del usuario.

## Outcome

El repositorio tiene cuatro agentes de proyecto descubribles por Codex. El chat
principal delega la coordinacion a `leader`; `explorer` investiga,
`implementer` modifica el alcance asignado y `reviewer` emite un veredicto
independiente. La evidencia vive bajo `docs/agent-reports/` y el chat transmite
solo sus rutas. Para comprobar que los TOML se cargan desde el inicio, debe
abrirse una tarea nueva de Codex despues de versionar estos archivos.
