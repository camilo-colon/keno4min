# Documentacion sencilla, coherente y evolutiva

Estado: completed
Creado: 2026-08-18
Responsable: leader

## Proposito

Una persona nueva puede orientarse desde `AGENTS.md` y `docs/README.md` sin
recorrer documentos redundantes, distinguir con claridad las fuentes de verdad
vigentes y ampliar la documentacion de forma incremental.

## Contexto

El repositorio tiene una base documental amplia para un producto que aun no
cuenta con paquetes Go compilables. El usuario pidio estudiar
`betta-tech/ejemplo-harness-subagentes` y adoptar solo sus practicas utiles para
mantener una estructura sencilla por ahora. Deben preservarse los cambios
existentes sin seguimiento y las decisiones reales ya registradas en
`ARCHITECTURE.md`, `docs/PRODUCT.md` y `docs/adr/`.

Evidencia delegada:

- `docs/agent-reports/2026-08-18-simplify-docs/exploration-reference.md`
- `docs/agent-reports/2026-08-18-simplify-docs/exploration-local.md`
- `docs/agent-reports/2026-08-18-simplify-docs/implementation-docs.md`
- `docs/agent-reports/2026-08-18-simplify-docs/review-docs.md`

## Criterios de aceptacion

- [x] `docs/README.md` ofrece una ruta de lectura corta y explica donde registrar cada tipo de conocimiento.
- [x] Los documentos conservados tienen una responsabilidad unica; no duplican instrucciones ni fuentes de verdad.
- [x] La organizacion toma practicas verificadas del repositorio de referencia sin copiar artefactos innecesarios.
- [x] Todos los enlaces Markdown locales y validaciones del harness pasan.
- [x] Una revision independiente emite `APPROVED` con evidencia en disco.

## Alcance y fuera de alcance

Incluye inventariar, consolidar, mover o retirar documentacion bajo `docs/`, y
ajustar indices o referencias directamente afectados. No incluye implementar
producto, inventar reglas de negocio, cambiar contratos HTTP ni redisenar la
arquitectura tecnica.

## Plan

1. Comparar en paralelo el repositorio de referencia y la documentacion local.
2. Decidir una estructura minima y registrar por que cada documento permanece.
3. Delegar la reorganizacion a un implementador con propiedad exclusiva.
4. Ejecutar revision independiente y corregir cualquier hallazgo.
5. Validar el harness, cerrar el plan y escribir el resumen de evidencia.

## Progress

- [2026-08-18 America/Bogota] Linea base verde con `./init.sh`; inventario inicial: 19 archivos bajo `docs/`.
- [2026-08-18 America/Bogota] Exploraciones externa y local completadas; ambas recomiendan una fuente por pregunta y no copiar `feature_list.json` ni `progress/`.
- [2026-08-18 America/Bogota] Implementacion consolidada y verificada; quedaron cinco documentos de primer nivel bajo `docs/`.
- [2026-08-18 America/Bogota] Revision independiente `APPROVED` y verificacion final verde; plan cerrado.

## Surprises & discoveries

- Todo el arbol aparece sin seguimiento porque el repositorio aun no tiene una linea base versionada; ningun archivo se tratara como descartable por ese motivo.
- El ejemplo remoto tiene solo tres documentos tematicos, pero repite el protocolo de agentes en varios archivos y presenta referencias inconsistentes; se adopta la separacion de responsabilidades, no la duplicacion.
- El repositorio local codifico siete documentos tematicos como obligatorios aunque aun no existen paquetes Go; seguridad, confiabilidad y puntaje de calidad contienen sobre todo estado futuro o informacion ya registrada en otras fuentes.
- Al mover el ultimo plan activo a `completed/`, `active/` queda vacio; se conserva `.gitkeep` para que el directorio requerido sobreviva un checkout limpio.

## Decision log

- [2026-08-18] Usar un unico implementador despues de explorar — la reorganizacion cruza indices y referencias, por lo que paralelizar escritura produciria solapamiento.
- [2026-08-18] Tratar el ejemplo como referencia, no plantilla literal — solo se adoptaran practicas que reduzcan complejidad en este repositorio.
- [2026-08-18] Conservar `PRODUCT.md`, `ENGINEERING.md`, `VERIFICATION.md` y `AGENT_WORKFLOWS.md` como documentos vivos — cada uno responde una pregunta distinta y `VERIFICATION.md` replica la separacion util del ejemplo.
- [2026-08-18] Retirar por ahora `SECURITY.md`, `RELIABILITY.md` y `QUALITY_SCORE.md` despues de preservar hechos unicos — podran reaparecer cuando existan controles, runtime o metricas concretas.
- [2026-08-18] Usar `docs/README.md` como unico indice — los README de colecciones y el tracker de deuda se consolidan para evitar navegacion y estados duplicados.

## Validation

| Comando o prueba | Resultado | Fecha |
| --- | --- | --- |
| `./init.sh` (linea base) | PASS; sin paquetes Go compilables | 2026-08-18 |
| Escaneo de enlaces Markdown locales | PASS; cero destinos rotos | 2026-08-18 |
| `bash -n scripts/harness/check.sh init.sh` | PASS | 2026-08-18 |
| Revision independiente | APPROVED; `review-docs.md` | 2026-08-18 |
| `./init.sh` (final) | PASS; harness verde, sin paquetes Go compilables | 2026-08-18 |

## Recuperacion

Los cambios son solo archivos de documentacion sin seguimiento. Ante un fallo,
se corrigen de forma incremental conservando el contenido fuente; no se usaran
comandos destructivos ni se eliminaran decisiones sin reubicar su informacion.

## Outcome

La ruta principal quedo reducida a `README`, producto, ingenieria, verificacion
y flujo de agentes, con `docs/README.md` como unico indice. Se preservaron las
ADR, planes historicos y hechos unicos; se retiraron documentos prematuros e
indices redundantes. La estructura adopta divulgacion progresiva, verificacion
ejecutable y evidencia en disco del repositorio de referencia sin duplicar su
estado (`feature_list.json`/`progress`) ni su configuracion de Claude. No quedan
enlaces locales rotos ni hallazgos de revision. El riesgo conocido permanece:
las comprobaciones Go se omiten hasta que existan paquetes compilables y OpenAPI
aun necesita lint semantico antes del primer endpoint.
