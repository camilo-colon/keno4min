# 0022. Indexar tiquetes y persistir la fecha en milisegundos

- Estado: Aceptada
- Fecha: 2026-10-27
- Responsables: Equipo de cronos

## Contexto

Se necesitan consultas de tiquetes por TIR y por nodo con un rango de fechas.
El guardado actual solo ofrece claves por UUID y cupón. La fecha se persiste
como texto RFC 3339 de precisión variable. `GSI1` ya es compartido por usuarios,
nodos y draws, con ambas claves de tipo String.

## Decisión

Conservar las claves canónicas, la reserva de cupón y los dos `Put`
transaccionales. Agregar al tiquete `GSI1PK = TICKETS#TIR#<tir>` y
`GSI1SK = TICKET#<ticketId>` para el listado por TIR. Agregar
`GSI2PK = TICKETS#NODE#<nodeId>` y usar `createdAt` (Number) como sort key
del nuevo `GSI2`, con proyección `ALL`.

Persistir `CreatedAt.UnixMilli()` como `createdAt` (`int64`). El dominio
conserva `time.Time` y la respuesta HTTP conserva su fecha RFC 3339. Los
contratos futuros de lectura usarán `time.Time` con límites alineados a
milisegundos y el intervalo `[from, to)`.

El repositorio central de infraestructura administra el esquema. La proyección
de `GSI1` debe admitir los tiquetes conservando los atributos de sus demás
consumidores. El modelo de acceso documenta el contrato completo y la migración.

## Alternativas consideradas

- Fecha ISO con precisión fija en la sort key: válida, pero se elige el formato
  numérico para el rango temporal y la unidad explícita en el nombre.
- Fecha numérica en `GSI1SK`: incompatible con sus consumidores actuales, que
  usan String. Se conserva ese esquema y se agrega un índice separado.
- Referencias por TIR en la tabla base: permiten lectura fuerte, pero agregan
  escrituras y lecturas. Se aplazan hasta diseñar el acceso de liquidación.

## Consecuencias

- Los índices se mantienen automáticamente desde el tiquete canónico, con
  costo adicional de escritura y almacenamiento; la reserva no se indexa.
- Se descarta la precisión inferior a milisegundos en persistencia. Varios
  tiquetes pueden compartir la misma fecha numérica en `GSI2`; su orden relativo
  no está garantizado y el cursor deberá conservar la clave completa.
- Los datos anteriores requieren convertir la fecha y completar las claves
  antes de que los listados puedan incluirlos. Crear el índice no calcula
  esos atributos ausentes.
- Los listados son eventualmente consistentes. No garantizan la completitud
  necesaria para liquidar; la coordinación con el cierre sigue pendiente.
- Este cambio prepara el guardado. Las consultas, sus endpoints, el despliegue
  de los índices y la ejecución de la migración quedan pendientes.

## Referencias

- [Modelo de tiquetes](../data-model/ticket-access-patterns.md).
- [Tipos de datos de DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/HowItWorks.NamingRulesDataTypes.html).
- [Índices secundarios globales](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/GSI.html).
