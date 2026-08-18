# Cierre de la simplificacion documental

Fecha: 2026-08-18
Estado: completed

## Resultado

`docs/README.md` es ahora el unico indice y ofrece una ruta de lectura de tres
pasos. La documentacion viva de primer nivel quedo limitada a cinco fuentes con
responsabilidades distintas:

- `README.md`: orientacion y ubicacion del conocimiento;
- `PRODUCT.md`: hechos y decisiones de producto pendientes;
- `ENGINEERING.md`: convenciones de implementacion;
- `VERIFICATION.md`: comandos, evidencia y definicion de terminado;
- `AGENT_WORKFLOWS.md`: protocolo compartido de delegacion.

Las ADR, ExecPlans e informes permanecen como colecciones separadas. Se
preservaron completas las decisiones aceptadas y los planes historicos.

## Adaptacion del repositorio de referencia

Se estudiaron el arbol, `AGENTS.md`, los tres documentos tematicos,
`CHECKPOINTS.md`, `feature_list.json`, `progress/`, `init.sh` y los agentes de
`betta-tech/ejemplo-harness-subagentes`. Se adoptaron sus practicas utiles:

- divulgacion progresiva desde un mapa corto;
- una fuente por pregunta estable;
- criterios y verificacion ejecutables;
- separacion entre coordinacion, implementacion y revision;
- evidencia de agentes persistente en disco.

No se copiaron `feature_list.json`, `progress/`, hooks ni configuracion de
Claude. Los ExecPlans, informes por task-id y agentes Codex ya cubren esas
responsabilidades sin introducir una segunda fuente de estado.

## Consolidacion

El contenido vigente de seguridad, confiabilidad, calidad y deuda se traslado o
ya estaba representado en producto, arquitectura, ingenieria, verificacion y
checkpoints. Despues se retiraron los documentos prematuros y los indices de
coleccion redundantes. `AGENT_WORKFLOWS.md` se redujo al protocolo compartido y
enlaza las instrucciones particulares de `.codex/agents/*.toml`.

## Evidencia

- Exploracion externa: `exploration-reference.md`.
- Auditoria local: `exploration-local.md`.
- Implementacion: `implementation-docs.md`.
- Revision independiente: `review-docs.md`, veredicto `APPROVED`.
- Escaneo de enlaces Markdown: PASS, cero destinos rotos.
- `bash -n scripts/harness/check.sh init.sh`: PASS.
- `./init.sh`: PASS, `harness verde`.

## Riesgo residual

Todavia no existen paquetes Go compilables, por lo que `gofmt`, `go vet` y
`go test ./...` se omiten de forma visible. OpenAPI tiene validacion estructural,
pero requiere un linter semantico antes de publicar el primer endpoint. Todo el
arbol sigue sin seguimiento en Git, igual que en la linea base.
