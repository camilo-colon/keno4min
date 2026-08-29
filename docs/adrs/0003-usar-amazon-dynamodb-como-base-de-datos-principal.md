# 0003. Usar Amazon DynamoDB como base de datos principal

- Estado: Aceptada
- Fecha: 2026-08-28
- Responsables: Equipo de cronos

## Contexto

El sistema necesita una base de datos NoSQL administrada que mantenga una
latencia de lectura predecible ante un volumen alto y variable, pueda escalar
horizontalmente y reduzca la administración de infraestructura.

La solución debe integrarse de forma natural con los servicios serverless de
AWS y ofrecer un modelo de costos alineado con el consumo real.

## Decisión

Amazon DynamoDB será la base de datos principal del sistema.

Se elige por su rendimiento de baja latencia, escalado horizontal administrado,
opciones de capacidad basadas en el consumo e integración con servicios de AWS
como Lambda, DynamoDB Streams y Step Functions.

El modelo de datos se diseñará a partir de patrones de acceso conocidos. La
configuración de capacidad y los diseños de tablas, claves e índices se
definirán y evolucionarán según las necesidades y métricas de cada carga de
trabajo.

## Alternativas consideradas

### MongoDB

Se descartó por el mayor costo operativo esperado para el volumen proyectado y
por requerir más administración o servicios adicionales para integrarse con la
arquitectura serverless prevista.

### PostgreSQL

PostgreSQL es una alternativa sólida y no se descarta por una limitación general
del motor. Ofrece ventajas relevantes:

- un modelo relacional maduro, con integridad referencial, restricciones y
  transacciones ACID;
- consultas expresivas mediante SQL, uniones y agregaciones;
- flexibilidad para consultas ad hoc y un ecosistema amplio y portable.

No se elige como base de datos principal porque esas capacidades no son las que
más pesan en esta decisión. El sistema tiene patrones de acceso operacionales
conocidos y prioriza latencia predecible, crecimiento horizontal administrado y
una integración directa con cargas serverless y orientadas a eventos.

PostgreSQL puede escalar horizontalmente mediante réplicas, particionamiento,
sharding o servicios distribuidos compatibles. Sin embargo, según la modalidad
elegida, esto puede incorporar planificación de capacidad, administración de
conexiones y decisiones adicionales sobre distribución de datos. DynamoDB
incluye estas capacidades en el servicio y se ajusta mejor al perfil previsto,
sin que ello implique que sea la mejor opción para cualquier carga de trabajo.

## Consecuencias

### Ventajas

- El sistema obtiene baja latencia y escalado horizontal sin administrar
  servidores de base de datos.
- La integración con el ecosistema de AWS simplifica la construcción de flujos
  serverless y orientados a eventos.
- Los modos de capacidad permiten ajustar el consumo a cargas variables sin
  mantener capacidad de cómputo de base de datos permanentemente aprovisionada.
- Se reduce la carga operativa asociada con servidores, actualizaciones,
  disponibilidad y escalado de la base de datos.

### Costos y limitaciones aceptados

- El modelo deberá diseñarse alrededor de patrones de acceso conocidos y podrá
  requerir desnormalización y duplicación controlada de datos.
- Las consultas ad hoc, las uniones y las relaciones complejas serán menos
  naturales que en PostgreSQL; nuevos patrones de acceso pueden exigir índices
  o proyecciones adicionales.
- Las transacciones deberán mantenerse dentro de los límites y del modelo de
  acceso de DynamoDB; no se obtiene la misma flexibilidad relacional de
  PostgreSQL.
- El costo dependerá de las lecturas, escrituras, almacenamiento, índices y
  patrones de acceso. Los escaneos, índices innecesarios o claves mal
  distribuidas pueden degradar el rendimiento y aumentar el costo.
- La solución aumenta la dependencia tecnológica y operativa de AWS.

## Criterios de revisión

La decisión deberá revisarse si los patrones de consulta dejan de ser
predecibles, si predominan relaciones y transacciones complejas, si las
consultas ad hoc se vuelven un requisito principal o si las métricas demuestran
que otra tecnología ofrece una relación costo-beneficio superior.

## Referencias

- [Choosing an AWS database service](https://docs.aws.amazon.com/decision-guides/latest/decision-guides/databases-on-aws-how-to-choose.html)
- [What is Amazon DynamoDB?](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Introduction.html)
- [DynamoDB on-demand capacity mode](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/on-demand-capacity-mode.html)
- [Data modeling foundations in DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/data-modeling-foundations.html)
- [PostgreSQL transactions](https://www.postgresql.org/docs/current/tutorial-transactions.html)
- [PostgreSQL constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)
