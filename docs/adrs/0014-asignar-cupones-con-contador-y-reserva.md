# 0014. Asignar cupones con contador y reserva

- Estado: Reemplazada
- Fecha: 2026-10-20
- Responsables: Equipo de cronos
- Reemplazada por: [ADR 0015](0015-recibir-identificador-del-tiquete-desde-el-frontend.md)

## Contexto

La creación de tiquetes debe conservar la entrada compacta del legacy mediante
parámetros en la URL, omitiendo sus prefijos y el identificador del cliente.
El tiquete necesita una identidad interna UUID y un cupón numérico de 16 dígitos
para su operación por cajeros. Se aceptan saltos en la numeración, pero dos
tiquetes no pueden compartir cupón, incluso con ventas concurrentes.

## Decisión

Usar `GET /tickets` con `stv` y `sum` en la query y sin body ni ID en la ruta.
El servidor genera el UUID v7 del tiquete. La motivación del transporte es
conservar el formato solicitado para los dispositivos existentes; una mejora
de rendimiento requiere medición. Las respuestas usarán `Cache-Control: no-store`.

Asignar el cupón mediante un contador global atómico en DynamoDB, operado desde
una única región de escritura. El nuevo valor se obtiene de la respuesta del
incremento y se representa según el intervalo y formato del PRD. Se rechaza
una asignación que exceda el intervalo; el contador nunca se reinicia.

Guardar el tiquete y una reserva permanente por cupón en una misma transacción,
con condiciones de no existencia. La reserva enlaza el cupón con un único UUID
y sirve también para buscar el tiquete. El contador se incrementa antes y fuera
de esa transacción; se aceptan los huecos resultantes.

## Alternativas consideradas

- Cupón aleatorio con reserva condicional: evita un contador compartido, pero
  se elige la numeración consecutiva solicitada. La unicidad seguiría requiriendo
  una reserva en persistencia.
- Contador sin reserva: la asignación atómica evita carreras ordinarias, pero
  no impone por sí misma una restricción de unicidad al guardar los tiquetes.
- Secuencia sin saltos: no es un requisito y exigiría coordinación adicional.
- Body JSON para la creación: no conserva el formato de entrada acordado.

## Consecuencias

- La reserva impide aceptar cupones duplicados sin depender de una lectura
  previa ni de la unicidad de un índice secundario.
- Cada asignación requiere actualizar el contador; este ítem limita la capacidad
  global de asignación y debe validarse frente al volumen esperado.
- El cupón es predecible y revela el avance de las asignaciones. La autorización
  efectiva del cajero sigue siendo obligatoria.
- La creación por `GET` exige evitar cachés y activaciones por navegación o
  precarga; la venta se inicia por una acción explícita del cajero.
- La reserva no hace idempotente una venta ni su débito. El protocolo de
  idempotencia, la coordinación con backoffice y la respuesta HTTP quedan
  pendientes antes de implementar el flujo completo.
- El contrato y el modelo son decisiones de diseño; este ADR no implica que
  exista un endpoint de creación implementado.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Modelo de acceso de tiquetes](../data-model/ticket-access-patterns.md).
- [Autoincremento en DynamoDB](https://aws.amazon.com/blogs/database/implement-auto-increment-with-amazon-dynamodb/).
- [Unicidad mediante transacciones](https://aws.amazon.com/blogs/database/simulating-amazon-dynamodb-unique-constraints-using-transactions/).
