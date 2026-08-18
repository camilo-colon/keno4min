# Implementacion de la simplificacion documental

Fecha: 2026-08-18
Rol: `implementer`

## Resultado

La documentacion viva quedo reducida a cinco fuentes con responsabilidades
distintas: indice, producto, ingenieria, verificacion y coordinacion de agentes.
`docs/README.md` es el unico indice y enlaza directamente las dos ADR aceptadas,
el ciclo de ExecPlans y la ubicacion de informes.

## Cambios realizados

- `AGENTS.md`: mapa reducido a fuentes vigentes; el trabajo pendiente ahora se
  registra en un ExecPlan o issue concreto.
- `README.md`: introduccion breve al flujo Codex y enlace al protocolo, sin
  duplicar roles ni permisos.
- `CHECKPOINTS.md`: retiro de la dependencia del tracker de deuda.
- `docs/README.md`: ruta de lectura, tabla para ubicar conocimiento, indice de
  ADR y ciclo minimo de planes e informes.
- `docs/PRODUCT.md`: decision pendiente sobre confianza e identidad entre
  gateway y servicio.
- `docs/VERIFICATION.md`: limitacion explicita del check estructural de OpenAPI
  y evidencia requerida con omisiones y riesgo residual.
- `docs/AGENT_WORKFLOWS.md`: protocolo, propiedad, rutas y veredictos; las
  instrucciones detalladas se enlazan desde `.codex/agents/*.toml`.
- `scripts/harness/check.sh`: conjunto minimo de fuentes requeridas e indice de
  ADR validado desde `docs/README.md`.
- `docs/exec-plans/completed/2026-08-18-harness-engineering.md`: nota posterior
  que conserva el resultado historico y explica la consolidacion.

Se retiraron, despues de consolidar sus hechos vigentes:

- `docs/SECURITY.md`
- `docs/RELIABILITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/adr/README.md`
- `docs/agent-reports/README.md`
- `docs/exec-plans/README.md`
- `docs/exec-plans/tech-debt-tracker.md`
- `docs/exec-plans/active/.gitkeep`

Las ADR y los planes completados se preservaron completos; solo se agrego la
nota posterior indicada al plan historico afectado. No se modificaron `.codex/`,
arquitectura, contratos, codigo ni el ExecPlan activo.

## Verificacion

| Comando | Resultado |
| --- | --- |
| `./init.sh` antes de editar | PASS; sin paquetes Go compilables |
| `bash -n scripts/harness/check.sh init.sh` | PASS |
| escaneo Python de enlaces Markdown locales | PASS; cero destinos rotos |
| busqueda de referencias activas a archivos retirados | PASS; cero referencias en fuentes vivas y check |
| `./init.sh` despues de consolidar | PASS; sin paquetes Go compilables |

## Omisiones y riesgo residual

- `go vet` y `go test ./...` fueron omitidos automaticamente porque aun no hay
  paquetes Go compilables; `./init.sh` lo informa de forma explicita.
- OpenAPI solo tiene validacion estructural. El primer endpoint requiere fijar
  un linter semantico, ahora documentado en `docs/VERIFICATION.md`.
- Las menciones a archivos retirados permanecen unicamente en los dos informes
  de exploracion y en el ExecPlan activo como evidencia historica de esta misma
  tarea; no son enlaces ni fuentes documentales vigentes.
- Todo el arbol continua sin seguimiento en Git, igual que en la linea base; no
  se descarto ningun cambio ajeno.
