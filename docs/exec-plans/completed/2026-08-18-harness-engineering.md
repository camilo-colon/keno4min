# Harness de ingenieria verificable

Estado: completed
Creado: 2026-08-18
Responsable: agente implementador

## Proposito

Dejar Keno4min preparado para que humanos y agentes naveguen el mismo
conocimiento, ejecuten un flujo reproducible y evalúen cambios con criterios
objetivos.

## Contexto

El repositorio era un scaffold Go 1.26 sin commits ni codigo de negocio. Ya
contenia una estructura de paquetes, OpenAPI base y dos ADR aceptadas. Se
preservaron esas decisiones como fundamento en lugar de introducir una
arquitectura nueva.

## Criterios de aceptacion

- [x] `AGENTS.md` funciona como mapa breve con divulgacion progresiva.
- [x] Arquitectura, producto, seguridad, confiabilidad y verificacion tienen
  fuentes de verdad localizables.
- [x] El trabajo complejo tiene plantilla y ciclo de vida persistente.
- [x] Implementacion y revision pueden coordinarse mediante artefactos en disco.
- [x] Local y CI usan el mismo comando de verificacion.

## Alcance y fuera de alcance

Incluye documentacion del harness, checks estructurales y CI. No inventa reglas
de Keno, identidad, persistencia ni endpoints; esas brechas quedan registradas.

## Progress

- [2026-08-18] Auditados README, OpenAPI, ADR, estructura y toolchain.
- [2026-08-18] Creados mapa, documentos vivos, checkpoints y ExecPlans.
- [2026-08-18] Integrados checks locales y workflow de CI.

## Surprises & discoveries

- El repositorio no tenia commits y todos los archivos existentes estaban sin
  seguimiento; se trataron como trabajo del usuario y se conservaron.
- Go 1.26.4 ya estaba disponible localmente, consistente con `go.mod`.

## Decision log

- [2026-08-18] Usar ExecPlans en Markdown en vez de `feature_list.json`: no hay
  backlog confirmado y un inventario ficticio se volveria obsoleto.
- [2026-08-18] Mantener roles agnosticos del proveedor en docs: `AGENTS.md` puede
  ser usado por Codex u otro agente sin duplicar instrucciones por herramienta.
- [2026-08-18] Aplicar primero invariantes de alto valor con shell y Go; el lint
  semantico de OpenAPI queda pendiente hasta seleccionar una herramienta.

La decision de mantener roles solo agnosticos fue reemplazada despues de que el
usuario confirmara que el cliente objetivo es Codex. El seguimiento vive en
`2026-08-18-codex-custom-agents.md`.

## Validation

| Comando o prueba | Resultado | Fecha |
| --- | --- | --- |
| `./init.sh` | Verde | 2026-08-18 |
| `bash -n init.sh scripts/harness/check.sh` | Sintaxis valida | 2026-08-18 |
| Parseo YAML de OpenAPI y CI | Valido | 2026-08-18 |
| Escaneo de enlaces Markdown locales | Sin destinos rotos | 2026-08-18 |
| Escaneo de whitespace final | Sin hallazgos | 2026-08-18 |

## Recuperacion

Los cambios solo agregan documentacion y checks versionados. Se pueden revertir
por archivo; no hubo migraciones ni efectos externos.

## Outcome

El harness tiene un punto de entrada, conocimiento indexado, limites
arquitectonicos, planes persistentes, revision independiente y checks compartidos
con CI. Las brechas de producto y operacion quedan visibles en el tracker.

## Nota posterior

El 2026-08-18 la documentacion viva se simplifico a una fuente por pregunta. El
tracker y los documentos tematicos prematuros se consolidaron en producto,
ingenieria y verificacion; este plan se conserva sin reescribir como evidencia
del estado que existia al cerrarlo.
