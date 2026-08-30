# Patrones de acceso de tickets

| ID | Patrón de acceso | Destino | Operación | Partition key | Sort key o condición | Orden | Consistencia |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `AP-TICKET-01` | Obtener un ticket por ID | Tabla base | `GetItem` | `TICKET#<ticketId>` | `META` | No aplica | Fuerte |
| `AP-TICKET-02` | Obtener un ticket por cupón | Tabla base | Dos `GetItem` | `COUPON#<coupon>` y luego `TICKET#<ticketId>` | `LOOKUP` y luego `META` | No aplica | Fuerte |
| `AP-TICKET-03` | Obtener los tickets de un cajero por rango de fecha de creación | `GSI1` | `Query` | `CASHIER#<cashierId>` | `CREATED#<createdAt>#TICKET#<ticketId>` entre `from` y `to` | Más reciente primero | Eventual |
| `AP-TICKET-04` | Obtener los tickets de un sorteo | Tabla base | `Query` por cada shard | `DRAW#<drawId>#SHARD#<n>` | Todos los ítems | Sin orden global | Fuerte |
| `AP-TICKET-05` | Obtener los tickets de un club para un sorteo | Tabla base | `Query` por cada shard | `DRAW#<drawId>#SHARD#<n>` | `begins_with(SK, "CLUB#<clubId>#")` | Por fecha de creación | Fuerte |
| `AP-TICKET-06` | Obtener los tickets pendientes de pago de un cajero por rango de fecha de creación | Tabla base | `Query` | `CASHIER#<cashierId>#PAYOUT_PENDING` | `CREATED#<createdAt>#TICKET#<ticketId>` entre `from` y `to` | Más reciente primero | Fuerte |

Todos los rangos de fecha son semiabiertos: `from <= createdAt < to`. Las
consultas que retornan varios ítems deben paginarse y ningún patrón requiere
`Scan` ni `FilterExpression`.
