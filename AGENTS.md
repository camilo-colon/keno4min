# Referencias del proyecto

| Cuándo consultar | Referencia |
| --- | --- |
| Alcance, requisitos y reglas de negocio | [PRD](docs/product-requirements.md) |
| Organización del código, límites de los slices y reglas de dependencia | [Guía de arquitectura](docs/architecture.md) |
| Contexto, justificación y consecuencias de decisiones arquitectónicamente significativas | [Registro de decisiones de arquitectura](docs/adrs/README.md) |

## Uso y mantenimiento

- Consulta únicamente la documentación necesaria para la tarea actual.
- Trata estas referencias como fuente de verdad y no dupliques su contenido.

### Arquitectura

- Consulta la guía antes de crear o mover paquetes, introducir un slice o cambiar
  la dirección de una dependencia.
- Actualízala cuando cambie la organización vigente del código o una regla de
  dependencia.

### Decisiones de arquitectura

- Consulta únicamente los ADRs directamente relacionados con la tarea.
- Crea un ADR para una decisión de impacto duradero que afecte la estructura,
  atributos de calidad, dependencias, interfaces o técnicas de construcción.
- No uses ADRs para detalles rutinarios de implementación o decisiones fáciles
  de revertir.
- No reescribas una decisión aceptada. Si cambia, crea un nuevo ADR, marca el
  anterior como `Reemplazada` y enlaza ambos documentos.
- Conserva cada ADR breve y sigue la convención y la plantilla del registro.

### Modelo de datos

- Si una tarea modifica el modelo de datos o un patrón de acceso, actualiza
  también la documentación correspondiente.
