# Auditoria local de documentacion

Fecha: 2026-08-18
Rol: `explorer`
Alcance: lectura de `AGENTS.md`, documentos raiz, todo `docs/` y
`scripts/harness/check.sh`. No se modificaron fuentes.

## Resultado ejecutivo

La documentacion contiene decisiones valiosas, pero la estructura viva es mayor
que el producto actual: existen 19 archivos bajo `docs/`, unas 1.159 lineas, y
todavia no hay paquetes Go compilables. La complejidad no proviene de las ADR ni
de los planes historicos, sino de varios documentos tematicos e indices que
repiten reglas, describen capacidades futuras o registran como deuda lo que en
esta etapa son decisiones de producto pendientes.

La estructura minima coherente para ahora es conservar cuatro documentos vivos
de primer nivel y tres colecciones de artefactos:

```text
docs/
  README.md
  PRODUCT.md
  ENGINEERING.md
  AGENT_WORKFLOWS.md
  adr/
    0001-package-organization.md
    0002-http-api-conventions.md
  exec-plans/
    TEMPLATE.md
    active/
    completed/
  agent-reports/
    <task-id>/
```

`docs/README.md` seria el unico indice. Producto, ingenieria y flujo de agentes
tendrian una responsabilidad distinta. Las ADR y los ExecPlans conservarian
decisiones e historia sin convertirse en lectura obligatoria para toda tarea.

## Evidencia observada

- `./init.sh` pasa; informa que aun no existen paquetes Go compilables.
- `scripts/harness/check.sh` exige actualmente siete documentos tematicos, tres
  README de colecciones y un tracker, de modo que la complejidad documental esta
  codificada como requisito aunque no exista comportamiento que la necesite.
- `docs/README.md` enumera estados y fechas de revision para siete documentos,
  pero no ofrece una secuencia corta de lectura ni explica con precision donde
  debe vivir conocimiento nuevo.
- `AGENTS.md`, `README.md`, `docs/AGENT_WORKFLOWS.md` y
  `docs/agent-reports/README.md` describen de forma repetida los mismos roles,
  rutas de informe y respuestas `done -> ...`.
- `docs/VERIFICATION.md`, `CHECKPOINTS.md`, `README.md`, `AGENTS.md` y el propio
  script repiten `./init.sh`, la evidencia requerida y la definicion de terminado.
- `docs/SECURITY.md` y `docs/RELIABILITY.md` declaran principalmente guardas
  genericas y trabajo futuro. Sus hechos unicos son el limite de confianza del
  gateway y la ausencia de runtime/SLO; ambos ya aparecen en
  `ARCHITECTURE.md`, `docs/PRODUCT.md` o el README raiz.
- `docs/QUALITY_SCORE.md` y `docs/exec-plans/tech-debt-tracker.md` describen las
  mismas ausencias desde dos mecanismos distintos. Varias entradas son estado
  esperado de un scaffold, no deuda producida por una implementacion.
- `docs/ENGINEERING.md` repite parte de `ARCHITECTURE.md` y ADR 0002, pero si
  conserva solo reglas de desarrollo, pruebas y verificacion aporta una fuente
  de verdad util.
- ADR 0001 y ADR 0002 contienen decisiones aceptadas y no deben resumirse ni
  retirarse durante esta reorganizacion.
- Los dos ExecPlans completados son historia verificable. No son documentacion
  viva y no deben borrarse por el hecho de simplificar el indice actual.
- `docs/exec-plans/active/.gitkeep` ya es innecesario porque el directorio
  contiene un plan real.

## Responsabilidad propuesta por documento conservado

| Fuente | Responsabilidad unica | Lectura |
| --- | --- | --- |
| `docs/README.md` | Ruta corta y regla para ubicar conocimiento | Primera visita |
| `docs/PRODUCT.md` | Hechos confirmados y decisiones de producto pendientes | Negocio o API |
| `docs/ENGINEERING.md` | Convenciones activas de implementacion, pruebas y comando canonico | Antes de editar codigo |
| `docs/AGENT_WORKFLOWS.md` | Orquestacion, propiedad de archivos e informes en disco | Trabajo delegado |
| `docs/adr/*.md` | Decisiones aceptadas, contexto y consecuencias | Solo cuando la tarea las toca |
| `docs/exec-plans/` | Plan vivo y registro historico de trabajos amplios | Tareas de varias etapas |
| `docs/agent-reports/<task-id>/` | Evidencia efimera/persistente de la delegacion | Durante una tarea multiagente |

`ARCHITECTURE.md` y `CHECKPOINTS.md` pueden permanecer en la raiz: el primero es
el mapa tecnico transversal y el segundo es la lista corta de revision. No hace
falta moverlos solo para que todo quede bajo `docs/`; hacerlo rompería rutas sin
reducir responsabilidades.

## Consolidacion recomendada

| Archivo actual | Accion sugerida | Contenido que debe preservarse |
| --- | --- | --- |
| `docs/VERIFICATION.md` | Consolidar en una seccion corta de `docs/ENGINEERING.md` y retirar | `./init.sh`, pruebas proporcionales al riesgo, evidencia exacta y checks omitidos |
| `docs/SECURITY.md` | Retirar despues de comprobar destinos | Limite gateway-servicio y decision de identidad en `PRODUCT.md`/`ARCHITECTURE.md`; secretos, entradas y logs ya estan en `AGENTS.md`/`CHECKPOINTS.md` |
| `docs/RELIABILITY.md` | Retirar por ahora | Cancelacion, atomicidad e idempotencia ya estan en arquitectura/ingenieria; crear un documento operacional cuando exista runtime o SLO real |
| `docs/QUALITY_SCORE.md` | Retirar sin migrar las calificaciones | No contiene decisiones; el estado real ya es observable en producto, codigo y planes |
| `docs/agent-reports/README.md` | Integrar el formato minimo en `docs/AGENT_WORKFLOWS.md` y retirar | Rutas unicas, propiedad y respuestas `done`, `blocked`, `APPROVED`, `CHANGES_REQUESTED` |
| `docs/exec-plans/README.md` | Integrar su ciclo de vida en `docs/README.md` o al inicio de `TEMPLATE.md` y retirar | Crear en `active/`, actualizar durante el trabajo y mover a `completed/` al cerrar |
| `docs/exec-plans/tech-debt-tracker.md` | Retirar | TD-001/003 pertenecen a decisiones pendientes de `PRODUCT.md`; TD-002 debe quedar como limitacion concreta en `ENGINEERING.md`; TD-004/005 son trabajo futuro, no deuda actual |
| `docs/adr/README.md` | Integrar el indice de las dos ADR y la regla de no reescritura en `docs/README.md`, luego retirar | Enlaces, estado de cada ADR y regla de reemplazo mediante una ADR nueva |
| `docs/exec-plans/active/.gitkeep` | Retirar | Nada; ya existe contenido real en la carpeta |

Esta consolidacion reduce ocho documentos estructurales o prematuros sin borrar
las dos ADR, los tres ExecPlans existentes ni la evidencia producida por
subagentes.

## Contenido que no debe perderse

1. La ausencia de contrato de identidad entre gateway y servicio debe figurar
   expresamente como decision pendiente en `docs/PRODUCT.md`; hoy esta mas
   desarrollada en `docs/SECURITY.md` que en el documento de producto.
2. El check OpenAPI actual es solo estructural. `docs/ENGINEERING.md` debe decir
   que se necesita lint semantico antes de publicar el primer endpoint.
3. La verificacion final debe seguir siendo `./init.sh`, y el informe/plan debe
   registrar comando, resultado, omisiones y riesgo residual.
4. Los documentos aceptados (`docs/adr/0001...` y `0002...`) deben conservarse
   completos. `docs/ENGINEERING.md` debe enlazarlos, no copiar sus reglas HTTP o
   de dependencias.
5. Los planes completados deben permanecer como historia. Si mencionan un
   archivo luego consolidado, conviene agregar una nota posterior de
   simplificacion en lugar de reescribir sus decisiones originales.

## Impacto exacto en referencias y checks

Si el lider adopta la estructura propuesta, un unico implementador debe ajustar
como minimo:

- `AGENTS.md`: reducir el mapa a los cuatro documentos vivos y colecciones;
  retirar referencias a `VERIFICATION.md`, `SECURITY.md`, `RELIABILITY.md` y
  `QUALITY_SCORE.md`; reemplazar el tracker como destino obligatorio de deuda.
- `README.md`: conservar solo una introduccion al flujo multiagente y enlazar
  `docs/AGENT_WORKFLOWS.md`; retirar la repeticion de roles, permisos y prompt.
- `CHECKPOINTS.md`: cambiar la referencia directa a
  `docs/exec-plans/tech-debt-tracker.md` por un ExecPlan o issue concreto.
- `docs/README.md`: convertirlo en ruta de lectura y unico indice; listar las
  dos ADR y explicar donde van producto, ingenieria, planes e informes.
- `docs/PRODUCT.md`: agregar de forma concisa el contrato de confianza/identidad
  pendiente antes de retirar `SECURITY.md`.
- `docs/ENGINEERING.md`: eliminar repeticiones de arquitectura/ADR y absorber
  verificacion mas la limitacion del lint OpenAPI.
- `docs/AGENT_WORKFLOWS.md`: absorber el formato y propiedad de informes;
  mantener un solo protocolo y remitir los detalles de cada rol a sus TOML.
- `docs/exec-plans/TEMPLATE.md`: solo si `docs/README.md` no conserva el ciclo
  completo, agregar alli las instrucciones minimas de apertura y cierre.
- `scripts/harness/check.sh`: dejar de exigir los ocho archivos retirados y
  cambiar el indice de ADR desde `docs/adr/README.md` a `docs/README.md`; seguir
  exigiendo las fuentes minimas y las carpetas `active/`/`completed/`.
- `docs/exec-plans/completed/2026-08-18-harness-engineering.md`: no reescribir la
  historia; agregar una nota posterior si el tracker y los documentos tematicos
  dejan de existir.
- `docs/exec-plans/active/2026-08-18-simplify-docs.md`: actualizarlo y moverlo al
  cierre corresponde al lider, no al implementador.

No hay referencias directas desde codigo de producto. Las definiciones
`.codex/agents/*.toml` solo necesitan `docs/AGENT_WORKFLOWS.md` y las rutas bajo
`docs/agent-reports/`, que se conservarian.

## Riesgos y secuencia segura

1. Consolidar primero el contenido unico en `PRODUCT.md`, `ENGINEERING.md`,
   `AGENT_WORKFLOWS.md` y `docs/README.md`.
2. Actualizar `AGENTS.md`, README raiz, `CHECKPOINTS.md` y el check mecanico.
3. Retirar solo entonces los documentos reemplazados y `.gitkeep`.
4. Buscar por cada basename retirado en todo el repositorio y validar enlaces
   Markdown locales.
5. Ejecutar `./init.sh`; el resultado final debe seguir verde.

El principal riesgo es borrar una guardia unica escondida en documentos
prematuros. Las cuatro preservaciones concretas anteriores evitan ese problema.
El segundo riesgo es crear otro documento agregador demasiado largo: cada
archivo vivo debe responder una sola pregunta y enlazar decisiones detalladas,
no copiarlas.

## Evaluacion contra el ExecPlan

La propuesta satisface el objetivo del plan porque produce una ruta breve desde
`AGENTS.md` a un solo indice, asigna una responsabilidad unica a cada documento
vivo y conserva las decisiones reales. Tambien mantiene la regla ya registrada
de tratar el repositorio externo como referencia, no como plantilla literal.

La decision final que corresponde al lider es el grado de consolidacion de los
tres README de colecciones. Para la etapa actual recomiendo retirarlos y usar
`docs/README.md` como unico indice; conservarlos mantendria una estructura valida
pero no cumpliria tan bien el pedido de comenzar sencillo.
