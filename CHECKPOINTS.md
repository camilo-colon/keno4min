# Checkpoints de estado final

Un implementador y un revisor usan estos criterios para evaluar el resultado,
independientemente de como se produjo.

## C1 — Intencion y alcance

- Los criterios de aceptacion son observables.
- El cambio resuelve una tarea coherente y no mezcla refactors ajenos.
- Las decisiones de negocio, seguridad o persistencia no se inventaron.
- Un trabajo complejo tiene un ExecPlan actualizado.

## C2 — Arquitectura y contratos

- Las dependencias respetan `ARCHITECTURE.md`.
- El negocio no conoce HTTP, MongoDB ni configuracion.
- Todo endpoint implementado existe primero en OpenAPI.
- Los tipos del transporte y de persistencia no sustituyen tipos de negocio por
  conveniencia.
- Una decision transversal nueva tiene ADR o una justificacion explicita.

## C3 — Correccion y pruebas

- El caso feliz y los fallos relevantes tienen pruebas.
- Las pruebas verifican comportamiento observable, no detalles accidentales.
- Los bordes externos validan entradas y cubren errores de traduccion.
- Concurrencia, reintentos e idempotencia tienen pruebas cuando aplican.
- `./init.sh` termina con codigo 0.

## C4 — Seguridad y confiabilidad

- No se confia en identidad o datos solo por haber llegado desde la red interna.
- Logs y errores no exponen secretos ni datos sensibles.
- Las operaciones de I/O respetan cancelacion y limites de tiempo definidos.
- Las escrituras criticas usan garantias atomicas adecuadas.
- Las migraciones son versionadas y tienen estrategia de recuperacion.

## C5 — Legibilidad para el siguiente agente

- Nombres y limites explican la responsabilidad sin depender del chat previo.
- Documentacion y ADR reflejan el comportamiento real.
- El ExecPlan registra decisiones, validacion y trabajo restante.
- El trabajo restante tiene un ExecPlan o issue concreto con impacto y evidencia
  de cierre.
- No quedan archivos temporales, debug ad hoc ni TODO sin contexto.

## C6 — Revision independiente

- El revisor inspecciono el diff completo y los riesgos, no solo el resumen.
- La evidencia incluye los comandos ejecutados y su resultado.
- Cualquier check omitido tiene razon y riesgo residual documentados.
- El estado declarado (`done`, parcial o bloqueado) coincide con la evidencia.
