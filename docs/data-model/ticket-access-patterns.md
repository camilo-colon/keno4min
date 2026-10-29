# Patrones de acceso de tiquetes

Estado: creación HTTP conectada al caso de uso, lectura del TIR desde runtime,
generación de cupones y guardado transaccional implementados. El listado por
nodo y rango de creación (`AP-TICKET-05`) está implementado y expuesto en
`GET /tickets`; el nodo se toma del claim `node_id` del token y un `nodeId`
en la query se ignora ([ADR 0026](../adrs/0026-filtrar-el-listado-de-tiquetes-por-el-club-del-token.md)). La consulta por ID
(`AP-TICKET-06`) está expuesta en `GET /tickets/:id` e incluye las bolas
copiadas al tiquete por la liquidación. Las demás lecturas de tiquetes y la aceptación financiera
siguen pendientes. La creación valida
estado y horario contra el draw canónico antes del cupón.
El guardado incluye las claves para consultar por TIR y por nodo y fecha, y
persiste la fecha como texto ISO 8601 en UTC con milisegundos. El despliegue de
los índices pertenece al repositorio central de infraestructura; consulta el
[ADR 0027](../adrs/0027-ordenar-gsi2-por-gsi2sk-con-fecha-iso.md).
Las reglas del producto y el
contrato de entrada viven en el [PRD, sección 7](../product-requirements.md#7-tiquetes-y-apuestas).
La decisión vigente está en el [ADR 0019](../adrs/0019-generar-identificador-del-tiquete-en-el-backend.md).

## Dominio de creación

La entidad es `Ticket` y cada elemento de `Bets` es una apuesta `Bet`. Sus
declaraciones y los valores de `Status` están implementados en
[`internal/tickets/ticket.go`](../../internal/tickets/ticket.go), que es la
referencia para los tipos Go. `Create(id, coupon, nodeID, tir, bets, createdBy)`
devuelve `(*Ticket, error)` y realiza las validaciones propias del dominio.
El constructor delega en el método privado `Ticket.validate() error`, que
valida los datos del tiquete y llama a la función privada `validateBets` para las apuestas. Un
error conserva el motivo del rechazo; la validación no modifica el tiquete.
El [adaptador HTTP](../../internal/tickets/adapters/http/create_ticket.go)
interpreta la query y ejecuta el caso
de uso. `ticketsfx.Module` conecta el flujo y registra `GET /tickets` en la
raíz. El contrato HTTP y su respuesta se definen en el PRD, sección 7.3.8.

`ID` es el UUID v7 generado por el backend. `NodeID` representa al club y se
copia del claim `node_id` del token; no se vuelve a resolver en consultas futuras.
Así se conserva el club de la venta si el usuario cambia de nodo. No se
introducen campos adicionales `CashierID`, `ClubID` o `AdminID` en esta
estructura.

`CreatedBy` se copia del claim `unique_name` del token al crear el tiquete,
conservando su escritura y mayúsculas; no se recibe como parámetro
independiente del frontend ni requiere una lectura. Es una copia histórica: un cambio posterior
del username no modifica los tiquetes existentes. El tiquete no incluye campos
`UserID` ni `Username`: el UUID del usuario no se persiste en el tiquete. `CreatedBy` contiene el nombre histórico
del vendedor y se utiliza para presentación; no es su UUID.

`TIR` asocia el tiquete al número de tiro global y único del sorteo, con el
mismo tipo `uint64` de `Draw.TIR`. El constructor exige un TIR positivo. El
caso de uso identifica el draw mediante `draws.Repository.GetRuntime` y lo
consulta por ID. Usa el TIR canónico después de comprobar que coincide con runtime. No se recibe como parámetro
independiente del frontend ni se genera desde el contador de cupones. El
tiquete no almacena un UUID adicional del sorteo.

La lectura del runtime solo identifica el sorteo. El caso de uso consulta
el draw canónico y valida estado y horario con `Draw.AcceptsTicketsAt` antes
de generar el cupón. Esta comprobación es previa al guardado. No se almacena la
moneda en el tiquete; su configuración sigue perteneciendo a backoffice.

El listado de tiquetes debe incluir el TIR y username almacenados en cada
tiquete para mostrar el número de tiro y el vendedor sin consultar un draw ni
un usuario por cada fila. Al definir el índice o la proyección de ese listado
se incluirán `tir` y `createdBy` entre sus atributos. El listado muestra el
username histórico de la venta, no el nombre actual consultado desde el usuario.
El endpoint, la paginación y el resto de la respuesta del listado siguen
pendientes.

`Create` exige UUID de tiquete y nodo distintos de cero, un TIR positivo,
`CreatedBy` no vacío ni compuesto solo por espacios y un cupón no vacío.
El caso de uso genera el UUID v7 del tiquete con `uuid.NewV7()`; el constructor
recibe valores `uuid.UUID` ya interpretados y solo comprueba que no sean nulos.

El cupón llega generado por el backend. Su rango y el padding mínimo de 4 dígitos son
responsabilidad del generador basado en el contador; `Create` solo exige que
el texto esté presente, sin volver a comprobar su longitud o contenido.
`Repository.GenerateCoupon` implementa el generador y se invoca desde el
caso de uso de creación antes de construir el tiquete.

El constructor exige una o más apuestas y valida las reglas de números y
valores de la sección 7.2 del PRD, sin límites monetarios por club. Los importes
usan `int64` y deben ser positivos. El total se obtiene sumando `ValueCents`;
no es un campo del dominio ni se persiste como atributo del tiquete. `Create`
mantiene una suma local únicamente para rechazar desbordamientos de `int64`.
El adaptador HTTP usa `strconv.ParseUint` con 8 bits antes de convertir a
`uint8`, impidiendo que una conversión oculte negativos o desbordamientos.
El constructor usa `validateBets` para exigir números entre `1` y `80`, valores
positivos y una suma sin desbordamiento; el adaptador no repite esa validación.

Se conserva el orden y la multiplicidad de las apuestas; cada `Bet` no requiere
un identificador propio. `Create` conserva los slices recibidos, sin copia
profunda. Los consumidores deben mantener las apuestas aceptadas y sus números
sin modificarlos. El guardado conserva las apuestas completas idénticas como
elementos separados de la lista.

`Create` fija `CreatedAt` con la hora UTC actual y comienza con `StatusPending`.
Si alguna validación falla, devuelve `(nil, error)`. Los demás
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
| `Coupon` | `coupon` | String numérico con padding mínimo de 4 dígitos |
| `CreatedBy` | `createdBy` | String, copia del claim `unique_name` del token en el momento de creación |
| `NodeID` | `nodeId` | String, UUID con guiones del club |
| `TIR` | `tir` | Number, entero positivo `uint64`, copiado del draw asociado |
| `Bets` | `bets` | List de Map; cada mapa contiene `nums` (List de Number) y `valueCents` (Number) |
| `Status` | `status` | String, valor del estado del dominio |
| `PrizeCents` | `prizeCents` | Number, premio normal del tiquete en centavos; `0` mientras no esté liquidado o si no tiene premio |
| `Balls` | `balls` | List de Number, bolas del draw escritas por la liquidación; ausente mientras el tiquete no esté liquidado |
| `CreatedAt` | `createdAt` | String, UTC con formato `2006-01-02T15:04:05.000Z07:00` |

Los UUID se almacenan en su representación canónica con guiones. El adaptador
usa `ticketItem`, `betItem` y `couponItem`, con etiquetas `dynamodbav`, y
`attributevalue.MarshalMap` del SDK oficial. `betItem.Nums` es `[]uint`, de modo
que `nums` se serializa como List de Number y no como el binario que produciría
`[]uint8`. Los valores monetarios se conservan como enteros `int64` y el TIR
como `uint64`, sin conversiones a coma flotante.
`CreatedAt` conserva su tipo `time.Time` en el dominio. El adaptador persiste
`createdAt` en UTC con milisegundos de ancho fijo, descartando la precisión
inferior al milisegundo sin modificar el objeto de dominio. El ancho fijo hace
que el orden lexicográfico coincida con el cronológico. La lectura reconstruye
el valor con `time.Parse(time.RFC3339Nano, createdAt)`, que también admite los
tiquetes guardados con precisión variable. La respuesta HTTP de creación
conserva `createdAt` en RFC 3339 y la precisión del objeto de dominio.
Las apuestas duplicadas conservan posiciones independientes en `bets`.
`value_cents` es el nombre conceptual usado en el PRD; `valueCents` corresponde
al atributo persistido. El JSON de respuesta HTTP se define en el PRD, sección 7.3.8.

`coupon` conserva los ceros iniciales. La reserva se comparte entre todos los
clubs, cajeros y sorteos; no incluye su identidad en la clave. Un cupón solo
puede apuntar a un tiquete. Las reservas son inmutables, no tienen TTL y no se
eliminan al anular un tiquete. No se requiere un GSI ni un `Scan` para garantizar
la unicidad o localizar un tiquete por cupón.

`ticketId` se genera en el caso de uso mediante `uuid.NewV7()` al crear el
tiquete. El frontend no envía ese identificador. Las claves canónicas y de
reserva se conservan.
Un nuevo UUID no identifica reintentos de una venta; la idempotencia sigue pendiente.

Los atributos adicionales para operaciones posteriores, el acceso para
liquidación y los registros de idempotencia y coordinación con
backoffice quedan pendientes.

## Índices de consulta

El ítem canónico incluye estos atributos adicionales. La reserva de cupón y el
contador no los incluyen, por lo que no participan en los índices de tiquetes.

| Atributo | Tipo | Valor |
| --- | --- | --- |
| `GSI1PK` | String | `TICKETS#TIR#<tir>`; TIR decimal sin padding |
| `GSI1SK` | String | `TICKET#<ticketId>` |
| `GSI2PK` | String | `TICKETS#NODE#<nodeId>` |
| `GSI2SK` | String | Mismo valor que `createdAt` |

El esquema a configurar en el repositorio central de infraestructura es:

| Índice | Partition key | Sort key | Proyección |
| --- | --- | --- | --- |
| `GSI1`, existente y compartido | `GSI1PK` (String) | `GSI1SK` (String) | Conservar los atributos de usuarios, nodos y draws; incluir además `ticketId`, `coupon`, `nodeId`, `tir`, `bets`, `status`, `prizeCents`, `createdBy` y `createdAt` |
| `GSI2`, desplegado | `GSI2PK` (String) | `GSI2SK` (String) | `ALL` |

Si `GSI1` ya usa `ALL`, su esquema y proyección no requieren cambios. Si usa
`INCLUDE`, verificar la unión de atributos requerida; cambiar la proyección de
un índice existente requiere recrearlo y coordinar sus consumidores. Las
claves `PK` y `SK` de la tabla se proyectan automáticamente. El prefijo
`TICKETS#` separa estas consultas de los demás tipos de ítem de `GSI1`.

`GSI1` ordena por UUID de tiquete; no ofrece un orden estricto por `CreatedAt`.
`GSI2` ordena por `GSI2SK`, cuyo ancho fijo en UTC mantiene el orden
cronológico y permite orden descendente. Varios
tiquetes pueden compartir un milisegundo: en un GSI se permiten claves
repetidas, pero el orden entre esos tiquetes no está garantizado. La paginación
deberá usar `LastEvaluatedKey` completo, incluidas las claves de la tabla, y no
un cursor basado únicamente en la fecha.

Las lecturas serán `Query` con condiciones sobre las claves, sin `Scan` ni
filtros para resolver TIR, nodo o rango. Para el rango `[from, to)`, el caso de
uso `usecase.List` recibe `time.Time` con precisión de milisegundos, exige
`from < to` y un límite entre 1 y 100 (50 por defecto). El adaptador consulta
`GSI2SK BETWEEN from AND to-1ms`, ambos con el formato de `createdAt`, porque `BETWEEN`
incluye ambos extremos, con `ScanIndexForward=false` para devolver primero los
tiquetes más recientes. El handler exige fechas RFC 3339 con zona y el caso de
uso rechaza precisión inferior al milisegundo en los límites. El cursor es el
`LastEvaluatedKey` completo en JSON codificado con base64 URL; el adaptador
no comprueba que pertenezca al mismo `GSI2PK`. El dominio continúa trabajando con
`time.Time`.

Ambos índices son eventualmente consistentes: un listado puede tardar en
reflejar una venta o cambio de estado. El listado por TIR no define un acceso
seguro para enumerar todas las ventas durante la liquidación. Esa operación
requiere diseñar la coordinación con el cierre y su garantía de completitud.
Los métodos de lectura y los endpoints todavía no están implementados.

La aplicación no crea índices ni modifica el esquema de la tabla al arrancar.

## Lectura del vendedor y club

La creación no lee la réplica de usuarios. Copia el claim `unique_name` del
token a `Ticket.CreatedBy` y `node_id` a `Ticket.NodeID`. El UUID del usuario no
se almacena en el tiquete. No se consulta el nodo para determinar el club. Las
comprobaciones de permisos, actividad y pertenencia del club están aplazadas en
esta etapa.

El [caso de uso `tickets/usecase.Create`](../../internal/tickets/usecase/create.go)
recibe `NodeID`, `CreatedBy` y `Bets`, sin depender de `users/ports`.
Consulta `draws/ports.Repository.GetCurrent`, que encapsula
`AP-DRAW-04` para resolver `Current.DrawId` y `AP-DRAW-01` para leer el draw
canónico. Usa su TIR y `Draw.AcceptsTicketsAt`, sin histórico ni GSI.
Un rechazo devuelve `ErrBettingClosed` (`409`) antes del cupón. Un error de
lectura/decodificación o un draw ausente devuelve `ErrDrawUnavailable` (`503`). No agrega otra
lectura o condición sobre el draw al guardar. Consulta el
[ADR 0021](../adrs/0021-validar-ventana-del-sorteo-al-crear-tiquetes.md).

## Patrones de acceso

`AP-TICKET-01` está implementado por `Repository.GenerateCoupon` y
`AP-TICKET-02` por `Repository.Save`, incluyendo los atributos de los índices.
`AP-TICKET-05` está implementado por `Repository.ListByNode` y `AP-TICKET-06`
por `Repository.GetByID`. `AP-TICKET-03` y `AP-TICKET-04` describen lecturas
previstas que todavía no están implementadas.

| ID | Operación | Acceso | Condición o consistencia |
| --- | --- | --- | --- |
| `AP-TICKET-01` | Asignar el siguiente consecutivo | `UpdateItem` en `COUNTER#TICKET` / `META` | Contador existente y dentro del intervalo; devuelve el nuevo valor |
| `AP-TICKET-02` | Guardar tiquete y reservar cupón | `TransactWriteItems`, con dos `Put` en el tiquete y su reserva | `attribute_not_exists(PK)` en cada `Put` |
| `AP-TICKET-03` | Resolver un tiquete por cupón | `GetItem` en `COUPON#<coupon>` / `META`; luego `GetItem` en `TICKET#<ticketId>` / `META` | Lectura fuerte en ambas operaciones |
| `AP-TICKET-04` | Listar tiquetes por TIR | `Query` en `GSI1`, `GSI1PK = TICKETS#TIR#<tir>` | Eventual; paginar hasta agotar resultados |
| `AP-TICKET-05` | Listar tiquetes por nodo y rango de creación | `Query` en `GSI2`, `GSI2PK = TICKETS#NODE#<nodeId>` y rango sobre `GSI2SK` | Eventual; rango `[from, to)` y cursor completo |
| `AP-TICKET-06` | Obtener un tiquete por ID | `GetItem` en `TICKET#<ticketId>` / `META` | Lectura fuerte |

`AP-TICKET-03` es un acceso interno previsto, no un endpoint de consulta ya
definido. Conocer el cupón no sustituye la autorización del cajero.

## Consulta por ID

`GET /tickets/:id` recibe el UUID del tiquete en la ruta y toma el club del
claim `node_id` del token. Un ID que no es UUID responde `400`.

`Repository.GetByID` implementa `AP-TICKET-06` con lectura fuerte, para que un
tiquete recién vendido sea visible de inmediato. Si el ítem no existe devuelve
`ports.ErrTicketNotFound`.

El [caso de uso `tickets/usecase.Get`](../../internal/tickets/usecase/get.go)
compara `Ticket.NodeID` con el club del token. Un tiquete de otro club devuelve
`ErrTicketNotFound`, igual que uno inexistente, y el handler responde `404` en
ambos casos para no revelar que el ID existe. La consulta es una sola lectura:
no consulta el draw.

`balls` sale del atributo `balls` del tiquete. La liquidación lo escribe junto
con `status` y `prizeCents`; Keno4min no lo escribe al crear el tiquete, por lo
que la respuesta es una lista vacía hasta que el tiquete se liquida. Como solo
se copia al liquidar, el resultado no se expone mientras la venta sigue abierta.
El adaptador guarda `balls` como List de Number mediante `[]uint` y lo omite si
está vacío. La lectura no vuelve a validar las bolas.
La respuesta tiene su propio tipo, separado del listado, y contiene los campos
del listado más `nodeId`, `bets` y `balls`. El listado omite `nodeId`, porque
siempre coincide con el club del token, y `bets`; conserva `totalCents`,
calculado a partir de las apuestas.

## Asignación del consecutivo

El puerto `tickets/ports.Repository` expone
`GenerateCoupon(context.Context) (string, error)`, implementado por el mismo
adaptador DynamoDB que realiza `Save`. Ambos métodos comparten cliente y tabla.

El contador se inicializa una única vez con `value = 0`, mediante un `PutItem`
condicionado a `attribute_not_exists(PK)`. Todas las instancias asignan valores
sobre el mismo ítem en una única región de escritura. No se crean contadores
independientes por instancia, club, cajero, sorteo o región.

El ítem inicial, en formato de atributos DynamoDB, es:

```json
{
  "PK": { "S": "COUNTER#TICKET" },
  "SK": { "S": "META" },
  "value": { "N": "0" }
}
```

Su creación forma parte de la preparación del entorno; este cambio no crea
datos en AWS ni agrega un inicializador. `GenerateCoupon` exige que exista y
no lo crea automáticamente ante un error.

Cada asignación ejecuta una única actualización atómica, sin lectura previa:

```text
UpdateExpression: ADD #value :one
ConditionExpression: attribute_exists(PK) AND #value >= :zero AND #value < :max
ReturnValues: UPDATED_NEW

#value = value
:zero  = 0
:one   = 1
:max   = límite máximo del contador definido en el PRD
```

Se utiliza exclusivamente el valor devuelto por esa operación para formar el
cupón, con `%04d` en Go: `1` produce `0001` y `10000` conserva sus cinco dígitos.
El padding no recorta el consecutivo ni define un máximo de `9999`.
`GenerateCoupon` exige que la respuesta incluya `value` como Number, lo
convierte a `uint64` sin coma flotante y comprueba el intervalo del PRD. Una
respuesta ausente, de otro tipo, fraccionaria o fuera del intervalo devuelve
un cupón vacío y un error. Los errores de DynamoDB, transporte y cancelación
del contexto se propagan conservando su causa. Si falta el contador o está
agotado, la condición impide incrementarlo; no se recrea ni reinicia.

El incremento queda fuera de la transacción que guarda el tiquete: su resultado
se necesita para construir la clave de la reserva. Por eso un fallo posterior
puede consumir un valor sin crear un tiquete. Un reintento del incremento también
puede consumir varios valores. No se decrementa el contador para recuperar esos
huecos y no se interpreta su valor como cantidad de ventas aceptadas.

## Reserva y persistencia

El puerto [`tickets/ports.Repository`](../../internal/tickets/ports/repository.go)
expone `GenerateCoupon(context.Context) (string, error)` y
`Save(context.Context, *tickets.Ticket) error`. El
[adaptador DynamoDB](../../internal/tickets/adapters/dynamodb/repository.go)
recibe el cliente compartido y `config.Config` mediante
`NewDynamodbRepository`, y utiliza `cfg.Dynamodb.TableName`.

Después de consultar el TIR actual, asignar el consecutivo y construir el tiquete
con las validaciones del dominio,
`AP-TICKET-02` guarda el tiquete canónico y la reserva en una misma transacción.
Ambos `Put` exigen que su clave no exista. Se guardan ambos registros o ninguno;
un cupón reservado nunca se reasigna ni se sobrescribe. La consulta previa de
existencia o un GSI por cupón no sustituyen esta condición.

`Save` recibe un tiquete ya preparado: serializa sus datos y el vínculo de
reserva, y realiza una única llamada `TransactWriteItems` con dos `Put` sobre
la tabla configurada. No ejecuta lecturas previas ni actualiza el contador.
Un argumento nulo devuelve un error antes de enviar solicitudes.

Una reserva ya existente bloquea el segundo tiquete y requiere tratar el
conflicto sin borrar la reserva ni reiniciar el contador. La condición de no
existencia del tiquete impide sobrescribir un UUID existente; no define una
respuesta idempotente. El tratamiento de un ID ya existente, los reintentos y
las respuestas inciertas queda pendiente de diseño.

Las cancelaciones, incluidos los conflictos de UUID o cupón, se devuelven como
errores y no se consideran un guardado exitoso. Los errores de DynamoDB y de
transporte se envuelven con el UUID del tiquete conservando su causa para
`errors.As` o `errors.Is`; no hay un segundo intento con escrituras
incondicionales. Se propaga el contexto del llamador y su cancelación.
El caso de uso propaga esos errores y devuelve un tiquete únicamente si `Save`
termina con éxito. El handler responde `500` ante errores de persistencia, sin exponer la causa interna.

El repositorio está conectado a Fx y al handler HTTP mediante `ticketsfx.Module`.
`GenerateCoupon` asigna el consecutivo y devuelve el cupón formateado;
`Save` recibe ese cupón ya asignado. `tickets/usecase.Create` coordina ambos
métodos: consulta el TIR actual, genera el cupón y el UUID v7, llama a `tickets.Create` con
el club y el username del token, y guarda el resultado. El constructor valida
las apuestas mediante `validateBets`, además de sus comprobaciones restantes.
Un rechazo posterior al incremento
puede dejar un salto permitido. No se recupera ni decrementa el
consecutivo. La aceptación financiera sigue pendiente.

La transacción descrita garantiza la relación exclusiva entre UUID y cupón.
No constituye por sí sola la aceptación financiera del tiquete. La fuente y
descuento de saldo siguen pendientes de diseño. La autenticación, las
comprobaciones del club y el tratamiento de idempotencia se aplazaron para
esta etapa. El estado y horario del sorteo se verifican antes del cupón usando
el draw canónico; esa validación no es atómica con la transacción de guardado.

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
