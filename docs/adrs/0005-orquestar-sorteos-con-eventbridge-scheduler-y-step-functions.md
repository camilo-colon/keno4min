# 0005. Orquestar sorteos con EventBridge Scheduler y Step Functions

- Estado: Aceptada
- Fecha: 2026-08-30
- Responsables: Equipo de cronos

## Contexto

Keno4min debe iniciar un sorteo global cada tres minutos, cerrar su ventana de
venta en un instante autoritativo, generar un solo resultado y liquidar los
tiquetes en el orden correcto. La entrega duplicada de un evento, los reintentos
o una reanudación no pueden duplicar el sorteo, reemplazar su resultado ni
liquidar nuevamente sus tiquetes.

El flujo actual necesita esperar 157 segundos entre el inicio del sorteo y su
cierre sin mantener un proceso de cómputo activo durante ese tiempo.

## Decisión

Amazon EventBridge Scheduler iniciará una ejecución de AWS Step Functions cada
tres minutos. La máquina de estados ejecutará secuencialmente:

1. `create_draw`: crea el sorteo del intervalo y registra `startsAt` y
   `closesAt`, con `closesAt = startsAt + 157 segundos`.
2. `wait_draw`: usa el valor de `closesAt` como timestamp absoluto de espera.
3. `draw`: genera y persiste definitivamente los 20 valores de `balls`.
4. `update_tickets`: procesa y liquida los tiquetes válidos del sorteo con el
   resultado persistido.

Cada tarea y escritura con efectos debe ser idempotente. Una entrega o ejecución
repetida para el mismo intervalo debe converger en el mismo sorteo, resultado y
liquidación.

## Alternativas consideradas

### Un proceso permanente con un temporizador interno

Se descartó porque obliga a operar y supervisar un proceso activo para mantener
la cadencia y complica la recuperación de una espera después de un reinicio.

### Una función que permanezca bloqueada durante la espera

Se descartó porque mantiene cómputo activo sin realizar trabajo y acopla la
duración de la espera a los límites de ejecución de la función.

### Eventos independientes para el cierre, resultado y liquidación

Se descartó por el momento porque distribuye el estado y el orden del flujo
entre varios disparadores. Puede reconsiderarse si la escala o el desacoplamiento
de la liquidación lo justifican.

## Consecuencias

### Ventajas

- La espera hasta `closesAt` es durable y no requiere cómputo activo.
- El orden del ciclo queda explícito y observable en una ejecución.
- Los reintentos y errores de cada etapa pueden administrarse de forma
  independiente sin perder el progreso del flujo.
- Los 23 segundos posteriores al cierre quedan disponibles como margen
  operativo antes del siguiente intervalo programado.

### Costos y riesgos aceptados

- El sistema depende de EventBridge Scheduler y Step Functions.
- La semántica de entrega y los reintentos exigen idempotencia en todas las
  tareas; Step Functions no la proporciona por sí solo para los efectos externos.
- `update_tickets` puede solaparse con la ejecución del siguiente intervalo si
  supera el margen nominal de 23 segundos. La política de concurrencia y atraso
  debe definirse antes de operar el flujo en producción.
- El anclaje del schedule, la zona horaria y el tratamiento de eventos tardíos
  siguen siendo decisiones pendientes del producto.

## Referencias

- [Product Requirements Document](../product-requirements.md#61-orquestación-del-ciclo)
- [Patrones de acceso de draws](../data-model/draw-access-patterns.md)
- [ADR 0003: Usar Amazon DynamoDB como base de datos principal](0003-usar-amazon-dynamodb-como-base-de-datos-principal.md)
