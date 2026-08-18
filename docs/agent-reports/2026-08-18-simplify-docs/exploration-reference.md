# Exploracion del repositorio de referencia

Fecha: 2026-08-18

Pregunta: que practicas documentales y de harness de
`betta-tech/ejemplo-harness-subagentes` conviene adoptar o evitar para mantener
este repositorio sencillo hoy y ampliable despues.

## Conclusion ejecutiva

Conviene adoptar los principios del ejemplo, no copiar su estructura de forma
literal. Su mayor acierto es que cada pieza responde una pregunta concreta:
`AGENTS.md` orienta, tres documentos pequenos definen arquitectura,
convenciones y verificacion, `init.sh` comprueba invariantes y los agentes dejan
evidencia en disco. Su mayor riesgo es la duplicacion del protocolo entre
`README.md`, `AGENTS.md`, `CLAUDE.md` y los agentes, que ya produjo referencias y
nombres de archivo inconsistentes.

Para este repositorio, la version minima recomendada es conservar una sola
fuente por tema y separar documentacion estable de evidencia operativa:

```text
AGENTS.md                    mapa y reglas no negociables
ARCHITECTURE.md              limites y dependencias del sistema
CHECKPOINTS.md               criterios de estado final
docs/
  README.md                  indice y ruta corta de lectura
  PRODUCT.md                 comportamiento y decisiones de producto
  ENGINEERING.md             convenciones tecnicas aplicables hoy
  VERIFICATION.md            comandos y evidencia exigida
  adr/                       solo decisiones aceptadas que necesitan historia
  exec-plans/                planes vivos de trabajos amplios
  agent-reports/             evidencia temporal por task-id
```

Los documentos `SECURITY.md`, `RELIABILITY.md`, `QUALITY_SCORE.md` y
`AGENT_WORKFLOWS.md` deberian mantenerse separados solo si el inventario local
demuestra que contienen reglas concretas y no repetidas. Si hoy son esqueletos o
repiten otros archivos, conviene mover su informacion unica a
`ENGINEERING.md`, `CHECKPOINTS.md`, `.codex/agents/` o al tracker de deuda y
retirar el duplicado. Se pueden volver a separar cuando el producto lo exija.

## Evidencia observada

### 1. El arbol real es deliberadamente pequeno

El arbol actual contiene en la raiz `.claude/`, `docs/`, `progress/`, `src/`,
`tests/`, `AGENTS.md`, `CHECKPOINTS.md`, `CLAUDE.md`, `README.md`,
`feature_list.json` e `init.sh`. Bajo `docs/` hay exactamente tres archivos:
`architecture.md`, `conventions.md` y `verification.md`. No hay un indice
adicional dentro de `docs/`; el `README.md` raiz presenta la estructura y el
arranque.

Fuentes:

- [arbol raiz](https://github.com/betta-tech/ejemplo-harness-subagentes/tree/main)
- [directorio docs](https://github.com/betta-tech/ejemplo-harness-subagentes/tree/main/docs)
- [README.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/README.md)

Esta estructura funciona porque el ejemplo tiene un dominio y una tecnologia
minimos. Es evidencia a favor de documentos con responsabilidad unica, no de
que todo repositorio real deba limitarse siempre a tres documentos.

### 2. `AGENTS.md` es un mapa, no la enciclopedia

El archivo declara divulgacion progresiva, define un arranque corto y enlaza la
fuente que se debe consultar para cada necesidad. Las reglas de arquitectura,
estilo y verificacion viven fuera del mapa. Tambien impone una sola unidad de
trabajo activa y exige mantener estado en disco.

Fuente: [AGENTS.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/AGENTS.md).

Practica demostrada: el punto de entrada debe decir donde buscar y que no se
puede violar; los detalles extensos pertenecen a la fuente tematica.

### 3. Los tres documentos tematicos tienen limites claros

- `docs/architecture.md` fija capas, dependencias permitidas, invariantes y
  anti-patrones estructurales. No explica el ciclo de agentes.
- `docs/conventions.md` fija estilo, nombres, estructura de archivos, pruebas y
  manejo de errores.
- `docs/verification.md` define niveles de prueba, comandos reproducibles y
  anti-patrones de validacion.

Fuentes:

- [docs/architecture.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/docs/architecture.md)
- [docs/conventions.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/docs/conventions.md)
- [docs/verification.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/docs/verification.md)

Practica demostrada: un documento debe responder una pregunta estable. En la
adaptacion local, `ENGINEERING.md` puede cumplir el papel de `conventions.md` y
`VERIFICATION.md` debe ser la unica fuente de comandos de prueba.

### 4. La coherencia se verifica con codigo

`init.sh` comprueba version de Python, presencia de los archivos base, estados
validos de `feature_list.json`, maximo una feature `in_progress` y la suite real
de pruebas. Acumula fallos y termina con un codigo de salida util para agentes y
CI.

Fuente: [init.sh](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/init.sh).

Practica demostrada: los enlaces y reglas esenciales del harness no deben
depender solo de prosa. El `init.sh` local debe comprobar la estructura que se
decida conservar; de otro modo, la reorganizacion documental se degradara sin
ser detectada.

### 5. El estado y los informes viven en disco

El ejemplo separa una plantilla de sesion activa (`progress/current.md`), una
bitacora append-only (`progress/history.md`) e informes de exploracion,
implementacion y revision. El historial registra agente, plan, archivos,
verificacion y cierre. El informe real de implementacion de `cli_edit` registra
archivos, decisiones y resultado de `init.sh`.

Fuentes:

- [directorio progress](https://github.com/betta-tech/ejemplo-harness-subagentes/tree/main/progress)
- [progress/current.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/progress/current.md)
- [progress/history.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/progress/history.md)
- [progress/impl_cli_edit.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/progress/impl_cli_edit.md)
- [progress/review_cli_edit.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/progress/review_cli_edit.md)

Practica demostrada: el chat no debe ser la unica memoria de coordinacion. La
adaptacion local por `task-id` es mas segura que un unico `current.md`, porque
evita que dos tareas o agentes escriban el mismo archivo.

### 6. Los roles tienen separacion de funciones

El arbol real define tres agentes personalizados: `leader`, `implementer` y
`reviewer`. El lider coordina y no implementa; el implementador cambia codigo y
pruebas; el revisor no edita y emite un veredicto. El lider puede recurrir a
exploradores acotados, aunque no existe un `explorer.md` personalizado en el
directorio del ejemplo.

Fuentes:

- [directorio .claude/agents](https://github.com/betta-tech/ejemplo-harness-subagentes/tree/main/.claude/agents)
- [.claude/agents/leader.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/agents/leader.md)
- [.claude/agents/implementer.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/agents/implementer.md)
- [.claude/agents/reviewer.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/agents/reviewer.md)

Practica demostrada: quien implementa no debe autoaprobarse. La sintaxis de
Claude no se debe copiar a Codex; solo se conserva la division de
responsabilidades.

### 7. Los criterios de cierre son objetivos

`CHECKPOINTS.md` convierte la salud final en una lista verificable: estructura
del harness, coherencia del estado, arquitectura, pruebas reales y cierre de
sesion. El reviewer recibe la obligacion de citar archivos y razones, no emitir
feedback generico.

Fuente: [CHECKPOINTS.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/CHECKPOINTS.md).

Practica demostrada: un checklist corto y observable sirve mejor que varias
politicas narrativas repetidas.

## Practicas que conviene adoptar ahora

### A. Un indice pequeno con rutas por intencion

`docs/README.md` deberia empezar con una ruta de lectura de tres pasos y luego
responder "donde escribo esto": producto, arquitectura, convenciones,
verificacion, decision historica, plan activo o informe de agente. No debe
resumir el contenido de cada documento en varios parrafos.

### B. Una fuente de verdad por pregunta

Asignacion recomendada:

| Pregunta | Fuente unica |
| --- | --- |
| Que hace el producto y que falta decidir | `docs/PRODUCT.md` |
| Cuales son los limites del sistema | `ARCHITECTURE.md` |
| Como se escribe codigo en este repositorio | `docs/ENGINEERING.md` |
| Como se demuestra que funciona | `docs/VERIFICATION.md` |
| Por que se tomo una decision duradera | `docs/adr/` |
| Que se esta haciendo en un trabajo amplio | `docs/exec-plans/active/` |
| Que produjo cada agente | `docs/agent-reports/<task-id>/` |
| Que significa estar listo | `CHECKPOINTS.md` |

Las referencias cruzadas deben enlazar la fuente, no volver a copiar sus
reglas.

### C. Documentacion estable separada de artefactos operativos

`PRODUCT.md`, `ARCHITECTURE.md`, `ENGINEERING.md`, `VERIFICATION.md` y las ADR
son conocimiento estable. ExecPlans e informes son evidencia de trabajo y
pueden crecer por tarea. Mezclarlos en un mismo nivel sin un indice crea la
sensacion de desorden que se quiere corregir.

### D. Evidencia breve, con propiedad exclusiva

Cada subagente debe escribir una ruta unica y devolver solo la referencia. Los
informes deben contener hallazgos destilados, archivos, comandos, resultado y
riesgo residual. No hace falta copiar la salida completa de cada test como hace
el informe `impl_cli_edit.md`; el comando, exit code y resumen son suficientes,
salvo que un fallo concreto requiera el fragmento de log.

### E. Verificar la topologia documental

El check local deberia comprobar, al menos:

1. archivos base requeridos;
2. enlaces Markdown locales;
3. TOML de agentes y nombres esperados;
4. scripts con sintaxis valida;
5. pruebas del proyecto cuando existan.

Esto adopta la intencion de `init.sh` sin atar el repositorio a
`feature_list.json`.

### F. Regla para crear un documento nuevo

Crear un archivo tematico solo cuando se cumplen las tres condiciones:

1. responde una pregunta distinta a las fuentes existentes;
2. contiene reglas o decisiones concretas que se consultan por separado;
3. tiene un propietario o evento claro que obliga a actualizarlo.

Si no se cumplen, el contenido debe ser una seccion de una fuente existente o
una entrada de deuda. Esta regla permite empezar pequeno y separar seguridad,
confiabilidad u operacion cuando el sistema realmente lo necesite.

## Practicas que conviene evitar o adaptar

### 1. No introducir `feature_list.json` junto a ExecPlans

En el ejemplo, `feature_list.json` es alcance, estado y criterios de aceptacion
de un producto didactico. `AGENTS.md`, `init.sh` y los agentes lo mutan.
Este repositorio ya usa ExecPlans con proposito, criterios, progreso, decisiones
y validacion. Introducir ambos generaria dos estados autoritativos y posibles
transiciones contradictorias.

Fuente: [feature_list.json](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/feature_list.json).

Recomendacion: el ExecPlan debe seguir siendo el plan vivo para trabajos
amplios; una tarea pequena puede conservar su plan en la conversacion.

### 2. No copiar `progress/current.md` e `history.md` literalmente

Un unico archivo mutable funciona en una demostracion de una feature a la vez,
pero crea contencion y mezcla sesiones en trabajo paralelo. El esquema local
`docs/agent-reports/<task-id>/` ya ofrece aislamiento y trazabilidad. El cierre
del ExecPlan sustituye la bitacora global.

### 3. No duplicar el protocolo en cuatro sitios

El ejemplo repite arranque y orquestacion en `README.md`, `AGENTS.md`,
`CLAUDE.md` y `.claude/agents/leader.md`. La repeticion ya presenta
inconsistencias concretas:

- `AGENTS.md` enlaza `scripts/demo_orchestration.py`, pero el arbol raiz actual
  no contiene un directorio `scripts/`.
- `README.md` documenta `progress/review_<feature>.md`; el reviewer prescribe
  `progress/review.md`; el arbol contiene `progress/review_cli_edit.md`.
- El lider dice que el lider lanza al reviewer, mientras el implementador dice
  que el propio implementador debe llamarlo, aunque su lista de herramientas no
  incluye `Agent`.
- El reviewer declara herramientas de solo lectura, pero su protocolo exige
  escribir `progress/review.md` y "marcar" checkpoints.

Fuentes: [AGENTS.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/AGENTS.md),
[README.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/README.md),
[leader.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/agents/leader.md),
[implementer.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/agents/implementer.md) y
[reviewer.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/agents/reviewer.md).

Recomendacion: `AGENTS.md` define el flujo general; cada archivo bajo
`.codex/agents/` define solo su rol; `docs/agent-reports/README.md` define rutas
y formato de evidencia. Cualquier documento humano adicional debe enlazar esos
tres lugares y no reescribirlos.

### 4. No copiar configuracion especifica de Claude

`CLAUDE.md`, frontmatter `tools:` y `.claude/settings.json` pertenecen al
runtime de Claude Code. Codex usa `.codex/config.toml` y
`.codex/agents/*.toml`. La practica portable es la separacion de roles y
permisos, no el formato.

Fuentes:

- [CLAUDE.md](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/CLAUDE.md)
- [.claude/settings.json](https://github.com/betta-tech/ejemplo-harness-subagentes/blob/main/.claude/settings.json)

### 5. No ejecutar toda la suite despues de cada escritura por defecto

El hook `PostToolUse` del ejemplo ejecuta tests tras cada Edit o Write y el hook
`Stop` vuelve a ejecutar `init.sh`. Esto es tolerable en un CLI de 22 pruebas,
pero escala mal y es especifico de Claude.

Recomendacion: pruebas dirigidas durante la implementacion y `./init.sh` al
inicio y al final. Automatizar hooks solo cuando el costo y los falsos positivos
esten medidos.

### 6. No copiar artefactos didacticos como requisitos universales

"Una feature a la vez", tres capas exactas, cero dependencias externas y tests
con filesystem real son decisiones correctas para Notes CLI, no principios
universales del harness. El equivalente local debe venir de arquitectura,
producto y riesgo reales.

### 7. No conservar documentos vacios por aspiracion

El ejemplo es valioso porque sus tres documentos tienen contenido ejecutable o
decisivo. Agregar desde hoy un archivo por cada futura disciplina aumenta rutas
sin aumentar conocimiento. La documentacion debe dividirse cuando el contenido
lo demande, no para anticipar una organizacion empresarial.

## Secuencia recomendada para esta reorganizacion

1. Inventariar cada afirmacion unica de los documentos actuales; no eliminar
   informacion solo porque el archivo parezca prematuro.
2. Consolidar conocimiento estable en `PRODUCT.md`, `ARCHITECTURE.md`,
   `ENGINEERING.md` y `VERIFICATION.md`.
3. Conservar ADR aceptadas y los ExecPlans/informes como historial, separados
   del camino de lectura principal.
4. Reducir `docs/README.md` a indice, ruta de lectura y regla de ubicacion.
5. Eliminar o convertir en redirect los documentos que queden sin
   responsabilidad unica, actualizando todas sus referencias en el mismo
   cambio.
6. Hacer que `./init.sh` compruebe la nueva estructura y los enlaces.
7. Revisar el resultado desde cero: una persona nueva debe llegar a la fuente
   correcta en uno o dos saltos desde `AGENTS.md` o `docs/README.md`.

## Riesgos y decisiones humanas

- Retirar `SECURITY.md` o `RELIABILITY.md` no debe significar perder limites de
  confianza, concurrencia o operacion. La informacion concreta debe migrarse
  antes; si es sustancial y se consulta de forma independiente, el documento
  debe permanecer.
- Las ADR aceptadas no deben reescribirse para aparentar una historia mas
  simple. El indice puede ocultarlas de la ruta inicial, pero siguen siendo
  evidencia de decisiones.
- Los informes y planes completados pueden crecer. No forman parte de la lectura
  inicial; se consultan por `task-id` o necesidad historica y no deben resumirse
  todos en `docs/README.md`.
- Si se conserva `AGENT_WORKFLOWS.md`, debe elegirse explicitamente como fuente
  humana de alto nivel y reducirse a enlaces y secuencia. No debe competir con
  las instrucciones ejecutables de `.codex/agents/`.

## Resultado recomendado

Adoptar del ejemplo: mapa progresivo, documentos con responsabilidad unica,
estado persistente por tarea, implementacion y revision separadas, criterios
objetivos y verificacion ejecutable.

No adoptar literalmente: `feature_list.json`, `progress/current.md`, bitacora
global, sintaxis de Claude, hooks en cada escritura, reglas de dominio del CLI
ni protocolos repetidos. La meta no es tener menos archivos a cualquier costo,
sino que cada archivo existente tenga una razon unica y comprobable para
existir.
