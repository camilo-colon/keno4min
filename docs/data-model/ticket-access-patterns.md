# Patrones de acceso de tiquetes

Estado: diseño acordado; sin implementación. Las reglas del producto y el
contrato de entrada viven en el [PRD, sección 7](../product-requirements.md#7-tiquetes-y-apuestas).
La decisión vigente está en el [ADR 0017](../adrs/0017-conservar-vendedor-como-created-by.md).

## Dominio de creación

La entidad es `Ticket` y cada elemento de `Bets` es una apuesta `Bet`. La
estructura acordada para creación es la siguiente; todavía no existen estos
tipos en el código:

```go
type Ticket struct {
    ID         uuid.UUID
    Coupon     string

    CreatedBy  string
    NodeID     uuid.UUID
    DrawID     uuid.UUID
    TIR        uint64

    Bets       []Bet
    TotalCents int64

    Status     Status
    CreatedAt  time.Time
}

type Bet struct {
    Nums       []uint8
    ValueCents int64
}

type Status string

const (
    StatusPending   Status = "pending"
    StatusCancelled Status = "cancelled"
    StatusLost      Status = "lost"
    StatusWinning   Status = "winning"
    StatusPaid      Status = "paid"
)
```

`ID` es el UUID v7 enviado como `id`. `NodeID` representa al club y se copia de
`user.NodeId`; no se vuelve a resolver desde el usuario en consultas futuras.
Así se conserva el club de la venta si el usuario cambia de nodo. No se
introducen campos adicionales `CashierID`, `ClubID` o `AdminID` en esta
estructura.

`CreatedBy` se copia de `user.Username` al crear el tiquete, conservando su
escritura y mayúsculas. El servidor lo obtiene de la misma lectura del usuario
por `userId`; no se recibe como parámetro independiente del frontend ni
requiere una lectura adicional. Es una copia histórica: un cambio posterior
del username no modifica los tiquetes existentes. El tiquete no incluye campos
`UserID` ni `Username`: `userId` se utiliza solo como entrada para consultar al
usuario y no se persiste en el tiquete. `CreatedBy` contiene el nombre histórico
del vendedor y se utiliza para presentación; no es su UUID.

`DrawID` identifica al sorteo y `TIR` conserva su número de tiro, con el mismo
tipo `uint64` de `Draw.TIR`. El servidor copia ambos del mismo draw asociado al
crear el tiquete; `TIR` debe ser positivo y permanece inmutable. No se recibe
como parámetro independiente del frontend ni se genera desde el contador de
cupones.

La resolución del draw para obtener `DrawID` y `TIR` sigue pendiente; ambos
campos permanecen en el modelo. La validación del estado y horario del sorteo
está aplazada según el flujo del PRD. No se almacena la moneda en el tiquete;
su configuración sigue perteneciendo a backoffice.

El listado de tiquetes debe incluir el TIR y username almacenados en cada
tiquete para mostrar el número de tiro y el vendedor sin consultar un draw ni
un usuario por cada fila. Al definir el índice o la proyección de ese listado
se incluirán `tir` y `createdBy` entre sus atributos. El listado muestra el
username histórico de la venta, no el nombre actual consultado desde el usuario.
El endpoint, la paginación y el resto de la respuesta del listado siguen
pendientes.

El constructor validará las apuestas y calculará `TotalCents` desde
`ValueCents`, sin recibir un total independiente del frontend. Los importes
usan `int64`, deben ser positivos y se suman detectando desbordamientos.
Los números se validan antes de convertirlos a `uint8`, para impedir que una
conversión oculte valores negativos o fuera de rango.

Se conserva el orden y la multiplicidad de las apuestas; cada `Bet` no requiere
un identificador propio. La construcción copia la lista de apuestas y los
números de cada una, de modo que una modificación posterior de los datos de
entrada no cambie el tiquete. Las apuestas aceptadas y sus valores permanecen
inmutables.

El servidor fija `CreatedAt` en UTC y comienza con `StatusPending`. Los demás
estados representan el ciclo funcional definido en el PRD; sus transiciones y
los datos adicionales de liquidación, jackpot, pago y anulación se definirán
al implementar esas operaciones.

No se incluye `RulesVersion`: la liquidación usa la tabla fija de pagos del
dominio. Un cambio futuro de esa tabla deberá preservar las reglas de tiquetes
anteriores antes de aplicarse. El dominio tampoco incluye claves, etiquetas ni
dependencias de DynamoDB o HTTP. El contador y la reserva son registros de
persistencia separados; `Ticket` solo contiene el cupón asignado.

## Ítems previstos

Los ítems usan la tabla configurada para Keno4min y sus claves `PK` y `SK`.

| Ítem | PK | SK | Contenido mínimo |
| --- | --- | --- | --- |
| Contador global | `COUNTER#TICKET` | `META` | `value`: Number, entero procesado como `uint64` |
| Tiquete canónico | `TICKET#<ticketId>` | `META` | `ticketId`: UUID v7 como String con guiones; `coupon`: String; datos de negocio exigidos por el PRD |
| Reserva y búsqueda por cupón | `COUPON#<coupon>` | `META` | `ticketId`: UUID del tiquete asociado |

### Atributos del tiquete canónico

| Campo de dominio | Atributo DynamoDB | Tipo DynamoDB |
| --- | --- | --- |
| `ID` | `ticketId` | String, UUID con guiones |
| `Coupon` | `coupon` | String de 16 dígitos |
| `CreatedBy` | `createdBy` | String, copia de `User.Username` en el momento de creación |
| `NodeID` | `nodeId` | String, UUID con guiones del club |
| `DrawID` | `drawId` | String, UUID con guiones |
| `TIR` | `tir` | Number, entero positivo `uint64`, copiado del draw asociado |
| `Bets` | `bets` | List de Map; cada mapa contiene `nums` (List de Number) y `valueCents` (Number) |
| `TotalCents` | `totalCents` | Number, entero `int64` |
| `Status` | `status` | String, valor del estado del dominio |
| `CreatedAt` | `createdAt` | String, RFC 3339 en UTC |

Los UUID se almacenan en su representación canónica con guiones. `nums` se
guarda como lista, no como conjunto: la serialización del adaptador debe
preservar un array numérico y no convertir `[]uint8` en un binario o base64.
Las apuestas duplicadas conservan posiciones independientes en `bets`.
`value_cents` es el nombre conceptual usado en el PRD; `valueCents` corresponde
al atributo persistido. No se define todavía el JSON de respuesta HTTP.

`coupon` conserva los ceros iniciales. La reserva se comparte entre todos los
clubs, cajeros y sorteos; no incluye su identidad en la clave. Un cupón solo
puede apuntar a un tiquete. Las reservas son inmutables, no tienen TTL y no se
eliminan al anular un tiquete. No se requiere un GSI ni un `Scan` para garantizar
la unicidad o localizar un tiquete por cupón.

`ticketId` procede del parámetro `id` enviado por el frontend. El servidor
valida el UUID v7 y lo conserva; no genera un identificador sustituto. Por
ahora no se define su uso como clave de idempotencia.

Los atributos adicionales para operaciones posteriores, los índices para
liquidación e históricos y los registros de idempotencia y coordinación con
backoffice quedan pendientes.

## Lectura del vendedor y club

La creación recibe `userId` y reutiliza `AP-USER-02`, una lectura fuerte por ID
definida en el [modelo de usuarios](user-access-patterns.md). Si el usuario no
existe, se rechaza la creación; los errores de lectura o deserialización se
propagan. Se copia `User.Username` a `Ticket.CreatedBy` y `User.NodeId` a
`Ticket.NodeID`. El UUID del usuario consultado no se almacena en el tiquete.
No se usa una búsqueda por username ni una consulta al nodo para determinar
el club. Las comprobaciones de autenticación, permisos, actividad y pertenencia
del club están aplazadas en esta etapa.

## Patrones previstos

| ID | Operación | Acceso | Condición o consistencia |
| --- | --- | --- | --- |
| `AP-TICKET-01` | Asignar el siguiente consecutivo | `UpdateItem` en `COUNTER#TICKET` / `META` | Contador existente y dentro del intervalo; devuelve el nuevo valor |
| `AP-TICKET-02` | Guardar tiquete y reservar cupón | `TransactWriteItems`, con dos `Put` en el tiquete y su reserva | `attribute_not_exists(PK)` en cada `Put` |
| `AP-TICKET-03` | Resolver un tiquete por cupón | `GetItem` en `COUPON#<coupon>` / `META`; luego `GetItem` en `TICKET#<ticketId>` / `META` | Lectura fuerte en ambas operaciones |

`AP-TICKET-03` es un acceso interno previsto, no un endpoint de consulta ya
definido. Conocer el cupón no sustituye la autorización del cajero.

## Asignación del consecutivo

El contador se inicializa una única vez con `value = 0`, mediante un `PutItem`
condicionado a `attribute_not_exists(PK)`. Todas las instancias asignan valores
sobre el mismo ítem en una única región de escritura. No se crean contadores
independientes por instancia, club, cajero, sorteo o región.

Cada asignación ejecuta una única actualización atómica, sin lectura previa:

```text
UpdateExpression: ADD #value :one
ConditionExpression: attribute_exists(PK) AND #value >= :zero AND #value < :max
ReturnValues: UPDATED_NEW

#value = value
:zero  = 0
:one   = 1
:max   = límite de 16 dígitos definido en el PRD
```

Se utiliza exclusivamente el valor devuelto por esa operación para formar el
cupón, con `%016d` en Go. Si falta el contador o su valor es inválido o está
agotado, la asignación falla; el flujo de venta no lo recrea ni reinicia
automáticamente. No se usan tipos de coma flotante para convertir el contador.

El incremento queda fuera de la transacción que guarda el tiquete: su resultado
se necesita para construir la clave de la reserva. Por eso un fallo posterior
puede consumir un valor sin crear un tiquete. Un reintento del incremento también
puede consumir varios valores. No se decrementa el contador para recuperar esos
huecos y no se interpreta su valor como cantidad de ventas aceptadas.

## Reserva y persistencia

Después de validar la entrada, consultar el usuario, validar las apuestas y
asignar el consecutivo conforme al flujo del PRD,
`AP-TICKET-02` guarda el tiquete canónico y la reserva en una misma transacción.
Ambos `Put` exigen que su clave no exista. Se guardan ambos registros o ninguno;
un cupón reservado nunca se reasigna ni se sobrescribe. La consulta previa de
existencia o un GSI por cupón no sustituyen esta condición.

Una reserva ya existente bloquea el segundo tiquete y requiere tratar el
conflicto sin borrar la reserva ni reiniciar el contador. La condición de no
existencia del tiquete impide sobrescribir un UUID existente; no define una
respuesta idempotente. El tratamiento de un ID ya existente, los reintentos y
las respuestas inciertas queda pendiente de diseño.

La transacción descrita garantiza la relación exclusiva entre UUID y cupón.
No constituye por sí sola la aceptación financiera del tiquete. La fuente y
descuento de saldo siguen pendientes de diseño. La autenticación, las
comprobaciones del club, la validación del estado y horario del sorteo y el
tratamiento de idempotencia se aplazaron para esta etapa; no se consideran
resueltos por persistir los dos registros. La resolución de `DrawID` y `TIR`
desde el mismo draw también queda pendiente para completar la construcción.

## Límites operativos

- El contador concentra las asignaciones en un solo ítem. Su capacidad y latencia
  deben verificarse con el volumen esperado antes del despliegue.
- La restauración de datos debe conservar contador, tiquetes y reservas sin
  reutilizar cupones previamente emitidos. Su procedimiento operativo queda
  pendiente; no se habilitan ventas reiniciando el contador.
- El cupón es predecible. La autorización de consulta y pago depende de la
  identidad y los permisos del cajero, no de mantener secreto el consecutivo.
- El tiquete conserva el nombre del vendedor sin su UUID. La correlación con
  el vendedor para consultas y pagos sigue pendiente; no se presupone que el
  username sea único ni inmutable.

## Referencias

- [Autoincremento en DynamoDB](https://aws.amazon.com/blogs/database/implement-auto-increment-with-amazon-dynamodb/).
- [Unicidad mediante transacciones](https://aws.amazon.com/blogs/database/simulating-amazon-dynamodb-unique-constraints-using-transactions/).
