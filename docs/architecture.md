# Arquitectura

Keno4min se organiza por funcionalidad. Cada módulo contiene su dominio y casos
de uso, con adaptadores separados para HTTP y DynamoDB. Consulta el
[ADR 0008](adrs/0008-organizar-por-funcionalidad-y-adaptadores.md).

## Estructura

```text
cmd/api/                     # Arranque HTTP e inyección de dependencias
cmd/worker/                  # Arranque Fx de consumidores SQS y suscriptores
cmd/script/                  # Generador del módulo Terraform de suscripciones
infra/eventbridge/           # Proyecto Terraform: provider, variables y estado
infra/eventbridge/generated/ # Módulo regenerado por cmd/script, ignorado por Git
internal/
├── config/                  # Variables de entorno
├── platform/aws/            # Configuración base del SDK AWS y módulo Fx
├── platform/dynamodb/       # Cliente DynamoDB y módulo Fx
├── platform/server/         # Servidor HTTP y middleware transversal
├── platform/auth/           # Middleware Bearer: Identity del token, New y FromContext
├── platform/workerfx/       # Consumidores SQS, deserialización y ciclo de vida Fx
├── users/
│   ├── user.go              # Réplica del usuario de backoffice
│   ├── events/              # Contrato UserCreatedDomainEvent
│   ├── ports/               # Contrato de persistencia de usuarios
│   ├── usecase/             # Creación idempotente por ID
│   ├── subscribers/         # Adaptación del evento al caso de uso
│   ├── adapters/dynamodb/   # Lectura y escritura condicional
│   └── usersfx/             # Repositorio compartido; worker y generador
├── tickets/
│   ├── ticket.go            # Entidad Ticket, apuesta Bet, estados y constructor Create
│   ├── ports/
│   │   └── repository.go    # Puerto de generación de cupones y guardado de tiquetes
│   ├── usecase/
│   │   └── create.go        # Generación del UUID y cupón, creación
│   ├── ticketsfx/           # Registro Fx de creación y ruta raíz
│   └── adapters/
│       ├── dynamodb/
│       │   ├── item.go      # Formato persistido de tiquete, apuestas y reserva
│       │   └── repository.go # GenerateCoupon atómico y Save transaccional
│       └── http/
│           ├── create_ticket.go # Parseo, validaciones y respuesta JSON de creación
│           └── routes.go    # GET /tickets, registrado en el router raíz
└── draws/
    ├── draw.go              # Entidad y reglas de negocio
    ├── ports/               # Contrato de runtime e histórico de Draw
    ├── usecase/             # GetRuntime y composición del histórico
    ├── adapters/
    │   ├── dynamodb/        # Lectura de runtime y GSI1; conversión de ítems
    │   └── http/            # Ruta runtime y respuesta JSON
    └── drawsfx/             # Registro Fx consumido por cmd/api
```

Las pruebas viven junto al código que verifican. Los paquetes y archivos se
crean cuando se implementa una responsabilidad, sin anticipar operaciones.

## Dependencias

- `draws` contiene la entidad `Draw` y sus invariantes. No importa Fiber, AWS,
  Fx, sus puertos ni sus adaptadores.
- `draws/ports` define el contrato de persistencia y las proyecciones de runtime;
  depende de `draws` para devolver entidades `Draw` por ID y en el histórico.
- `draws/usecase` consume el puerto y compone la respuesta de runtime.
- `draws/adapters/http` depende de los casos de uso y Fiber; serializa la
  respuesta con bolas como arrays numéricos.
- `draws/adapters/dynamodb` depende del dominio, sus puertos y el SDK AWS.
  Construye claves, lee y convierte los datos almacenados.
- `draws/drawsfx` registra el repositorio, caso de uso, handler y rutas. Esta
  composición se mantiene fuera del dominio, como en `usersfx` y `nodesfx`,
  para que los puertos y adaptadores puedan importar `Draw` sin ciclos.
  Consulta el [ADR 0018](adrs/0018-separar-composicion-fx-del-dominio-draw.md).
- `platform/auth` sigue el patrón de los middlewares de Fiber, como
  `requestid`: `New` decodifica el JWT sin verificarlo, porque el API Gateway lo
  valida, y guarda una `Identity` con `fiber.StoreInContext` bajo una clave
  privada. Los handlers la leen con `auth.FromContext(ctx)`; gracias a
  `PassLocalsToContext` en `serverfx`, también está disponible en
  `ctx.Context()`. Los casos de uso reciben los valores que necesitan como
  parámetros y no dependen de este paquete. La autenticación pertenece al
  backoffice según el PRD; por eso este paquete vive en `platform` y no como un
  módulo de negocio. `serverfx` monta el middleware en todas las rutas. Consulta el
  [ADR 0023](adrs/0023-delegar-la-validacion-del-token-al-api-gateway.md).
- Los adaptadores no se importan entre sí. Un adaptador puede implementar varios
  contratos pequeños; no se introduce un repositorio CRUD genérico.
- `tickets` contiene `Ticket`, `Bet`, `Status` y el constructor `Create` con sus
  validaciones de dominio. Depende únicamente de la biblioteca estándar, sin
  importar HTTP ni persistencia.
- `tickets/adapters/http` depende de Fiber, de `platform/auth`, del caso de uso
  y del dominio.
  Toma el club y el vendedor de `auth.Identity`, interpreta la query y serializa el tiquete
  guardado con apuestas como arrays numéricos.
- `tickets/ports` define `Repository.GenerateCoupon` y `Repository.Save`, y
  depende del dominio `tickets` y `context`. `tickets/adapters/dynamodb`
  implementa ese contrato usando el SDK
  AWS y la configuración de la tabla. El formato persistido permanece en el
  adaptador; no se agregan etiquetas de almacenamiento al dominio.
- `tickets/usecase` depende del dominio `tickets`, de `tickets/ports` y
  `draws/ports`. Recibe el club y el vendedor como parámetros y obtiene el draw
  mediante `GetCurrent`; no importa adaptadores, HTTP, AWS ni `users`.
- `tickets/ticketsfx` compone repositorio, caso de uso, handler y rutas.
  `cmd/api` agrega este módulo sin `usersfx`; el repositorio de usuarios solo
  se carga en el worker y el generador.
- `serverfx` registra el grupo Fx `rootRoutes` en la raíz y conserva `routes`
  bajo `/v1`. Consulta el [ADR 0020](adrs/0020-conectar-creacion-http-de-tiquetes.md).
- `cmd/api` construye el cliente AWS, el store, el caso de uso y el handler, y
  registra la ruta. `platform` comparte infraestructura técnica, no negocio.

### Configuración compartida de AWS

`platform/aws` proporciona un único `aws.Config` por aplicación Fx mediante
`config.LoadDefaultConfig`, con un plazo de carga de diez segundos. El SDK
resuelve región y credenciales desde sus fuentes habituales, incluidas las
variables `AWS_REGION`, `AWS_PROFILE` y los roles del entorno de ejecución.

`platform/dynamodb` recibe esa configuración base y `config.Config` para
construir el cliente. Aplica `DYNAMODB_ENDPOINT` únicamente al cliente DynamoDB;
los futuros clientes AWS podrán consumir la misma configuración base.

`cmd/api` registra ambos módulos junto a los de funcionalidad y servidor. El
repositorio de `draw` recibe el cliente y `config.Config`, y conserva el nombre
de la tabla definido por `DYNAMODB_TABLE`. La configuración compartida de AWS
no contiene nombres de tablas ni reglas de persistencia del negocio.

## Consulta implementada

`GET /api/v1/draws/{id}` ejecuta `draw.GetByID`. El caso de uso valida el parámetro,
lee mediante `IDReader`, invoca `Draw.Validate()` y comprueba que el draw tenga
el ID solicitado. Devuelve la entidad validada al adaptador HTTP.

`Draw.Validate()` centraliza las reglas de identidad, tiempos, estado y bolas.
`draw.New` usa la misma validación al crear un sorteo. Los errores de datos
almacenados se distinguen de los errores de parámetros de la petición.

La consulta por TIR anterior fue retirada y conserva su diseño de acceso
documentado. `draws/ports.Repository.GetLatestDraws` devuelve hasta diez
entidades `draw.Draw`. El adaptador `draws/adapters/dynamodb` consulta `GSI1`
con `GSI1PK = DRAW#DRAWN` en orden descendente de `GSI1SK`, sin lecturas
adicionales por resultado. `draws/usecase.GetRuntime` consume esta consulta
después de leer el runtime y mapea cada resultado a `history` con TIR y bolas,
conservando el orden del repositorio. `GET /v1/draws/runtime` incluye ese histórico
y propaga como error interno el fallo de cualquiera de las lecturas.
`draws/adapters/dynamodb.GetRuntime` lee la proyección `DRAW_RUNTIME/META`
mediante un `GetItem` fuertemente consistente y la convierte a `ports.Runtime`.
Las futuras consultas del draw completo pueden reutilizar el dominio y la
persistencia; las proyecciones parciales mantienen sus contratos específicos.

Los endpoints de sorteos son de lectura. Las tareas coordinadas por Step
Functions producirán el estado canónico y sus proyecciones; el arranque de
worker registra la configuración AWS, los clientes DynamoDB y SQS, los
suscriptores de nodos y usuarios, y `workerfx.Module`. Ejecuta `fx.App.Run()` para iniciar
los consumidores y coordinar su cierre mediante los hooks del ciclo de vida.
Cada consumidor deserializa y ejecuta únicamente su suscriptor; elimina el
mensaje de SQS después de que el manejador termine correctamente. Los mensajes
inválidos o con errores de procesamiento permanecen para reentrega y la DLQ
configurada, sin impedir que se procese el resto del lote. Un error de consulta
de cola o recepción detiene ese consumidor y se registra; no hay reinicio
automático del consumidor. Los manejadores deben tolerar reentregas, incluso
si el procesamiento tuvo éxito y falló la eliminación del mensaje.

### Deserialización por suscriptor

`common.DomainEventSubscriber.SubscribeTo()` devuelve prototipos de eventos
como structs por valor. `Type()` sobre el prototipo declara un nombre exacto o
un patrón con un único `*` final. `workerfx.NewWorker` construye un
`DomainEventJSONDeserializer` por suscriptor y valida los registros antes de
iniciar los consumidores. `platform/workerfx` depende de los contratos de
`common`, no de los tipos concretos de `nodes/events`.

El deserializador lee el sobre EventBridge de `message.Body`, usa `detail-type`
como nombre autoritativo y deserializa únicamente `detail`. Prioriza el nombre
exacto; entre patrones coincidentes gana el prefijo más largo. Reflection crea
una instancia nueva por mensaje y el resultado se entrega por valor. El registro
no conserva ni reutiliza instancias de eventos. La conversión JSON no sustituye
la validación de campos obligatorios y reglas de negocio del manejador.

El deserializador convierte `detail` directamente al struct seleccionado, sin
añadir ni sobrescribir campos. Comprueba que el `Type()` del evento resultante
coincida con `detail-type`. Los eventos con comodín deben obtener su nombre
concreto del propio payload; no requieren un setter adicional en `common`. El
comodín selecciona el prototipo registrado; no deduce ni construye tipos
específicos desconocidos. Actualmente nodos solo registra
`NodeCreatedDomainEvent`; los prototipos con comodín existen únicamente en las
pruebas hasta que un caso de uso los requiera. Un manejador que registre varios
tipos debe aceptar todas sus posibles salidas. No existe un registro global
que altere esa selección.

`NodeCreatedSubscriber` usa `detail.nodeId` como identidad del nodo y mantiene
`detail.eventId` como identidad del evento. Adapta el evento a `CreateParams` y
delega en `nodes/usecase.Create`, registrado por `nodesfx.WorkerModule`.
`nodes/usecase` depende del dominio `nodes` y de `nodes/ports`, sin importar
eventos ni adaptadores. El caso de uso consulta primero el nodo por ID: si ya
existe, termina con éxito. Para un nodo nuevo con `ParentId`, consulta al padre
mediante `FindById`; si no existe, devuelve `ErrParentNotFound` y no guarda.
Los nodos sin padre no requieren esa segunda consulta. Los errores llegan al
consumidor para permitir reentregas, incluido el caso de un hijo que llega antes
que su padre. Esta comprobación es local a la réplica de nodos de Keno4min;
la administración de la jerarquía sigue perteneciendo al backoffice.

El adaptador DynamoDB de nodos
implementa `Save` con una escritura condicional: crea el nodo únicamente si no
existe. Un nodo existente se conserva y se considera procesado; los demás errores
se propagan para permitir la reentrega. Consulta el
[modelo de nodos](data-model/node-access-patterns.md), el
[ADR 0012](adrs/0012-deserializar-detail-sin-transformaciones.md) y el
[ADR 0013](adrs/0013-crear-nodos-solo-si-no-existen.md).

### Réplica de usuarios

`usersfx.WorkerModule` registra `UserCreatedSubscriber` en el grupo
`subscribers`. Escucha `cronos.core.user.created` y delega en
`users/usecase.Create`. La identidad es `detail.userId` (`UserID` en el evento); `detail.eventId`
identifica la entrega del evento y no forma parte de la clave del usuario.
El caso de uso exige un username no vacío y userId y nodeId distintos del UUID cero,
consulta por ID y termina con éxito si el usuario ya existe. Para uno
nuevo guarda `ID`, `Username`, `RoleId` y `NodeId` mediante una escritura condicional.
Los errores se propagan al consumidor para permitir reentregas.

`users/usecase` depende únicamente del dominio `users` y de `users/ports`;
el suscriptor adapta eventos y el adaptador DynamoDB implementa el puerto.
No se consulta la réplica de nodos durante la creación del usuario: se conserva
la asociación enviada por backoffice, sin imponer un orden entre ambas colas.
El módulo no administra autenticación ni permisos. Consulta el
[modelo de usuarios](data-model/user-access-patterns.md).

## Creación de tiquetes

`internal/tickets/ticket.go` implementa `Ticket`, `Bet`, sus estados y el
constructor `Create`. `tickets/usecase.Create` implementa la coordinación de
la validación del sorteo, la generación del UUID v7 y del cupón, y el guardado.
El contrato HTTP y las reglas acordadas están en el
[PRD, sección 7.3](product-requirements.md#73-creación).
El [modelo de tiquetes](data-model/ticket-access-patterns.md) define el dominio
`Ticket` y `Bet`, sus atributos persistidos, el contador global y la reserva
exclusiva del cupón. El [ADR 0025](adrs/0025-tomar-club-y-vendedor-de-los-claims-del-token.md)
registra la decisión vigente: el backend genera el UUID v7 y la creación toma del
token `node_id` como club de la venta y `unique_name` en `CreatedBy` como nombre
histórico del vendedor, sin consultar la réplica de usuarios. El tiquete no
conserva el UUID del usuario. El listado incluirá `createdBy` y TIR desde los datos del tiquete,
evitando consultas por cada fila.

El tiquete se asocia al sorteo mediante su TIR, sin guardar un UUID adicional
del draw. El total se deriva de las apuestas. `Create` valida los datos del
dominio y detecta desbordamientos de la suma local, conserva las apuestas y fija
el estado `pending` y la hora UTC de creación. Solo exige que el cupón no esté
vacío; su formato corresponde a `Repository.GenerateCoupon`.

El [flujo de la etapa actual](product-requirements.md#733-flujo-de-creación-de-la-etapa-actual)
aplaza autenticación, permisos, comprobaciones del club e idempotencia.
El saldo y su descuento siguen
pendientes. El handler toma el vendedor con `auth.FromContext`, recibe `stv` y
`sum`, sin body, e invoca el caso de uso. Comprueba la estructura de la query y la conversión de números
e importes antes de acceder a persistencia. Las reglas de las apuestas y el
desbordamiento del total se validan al construir el tiquete. Todas sus respuestas usan
`Cache-Control: no-store`. El manejo explícito de `HEAD` queda pendiente.

`ticketsfx.Module` registra `GET /tickets` en `rootRoutes`. La ruta queda en la
raíz y el runtime conserva `/v1/draws/runtime`. El contrato JSON, los códigos
HTTP se definen en el PRD, sección 7.3.8. Los límites técnicos configurables de
cantidad de apuestas y longitud de query quedan pendientes.

`usecase.NewCreate` recibe los repositorios de tiquetes y sorteos.
`Execute` recibe `CreateParams` con `NodeID`, `CreatedBy` y `Bets`, que el
handler completa desde `auth.Identity`. Obtiene el draw canónico con
`draws.Repository.GetCurrent` y llama a `Draw.AcceptsTicketsAt` con la hora UTC
del servidor antes del contador. El repositorio encapsula `GetRuntime` y
`FindByID` usando `Current.DrawId`, comprueba la referencia y coincidencia del
TIR, sin consultar el histórico. Un rechazo devuelve `ErrBettingClosed`;
el handler responde `409`. Los datos ausentes o inválidos devuelven `503`.
La comprobación se realiza antes de guardar, sin una condición sobre el draw
en la transacción. Consulta el
[ADR 0021](adrs/0021-validar-ventana-del-sorteo-al-crear-tiquetes.md).

Después genera cupón y UUID v7, construye con `tickets.Create` usando el nodo y
username recibidos, y guarda mediante `Save`. Solo devuelve el tiquete tras
un guardado exitoso. Propaga el contexto; los errores o la ausencia de draw en
`GetCurrent` se clasifican como `ErrDrawUnavailable`. El constructor mantiene sus
validaciones, aunque el adaptador ya haya validado las apuestas. Un rechazo
posterior al incremento puede dejar un salto permitido en el contador.

`tickets/ports.Repository` expone `GenerateCoupon(context.Context)
(string, error)` y `Save(context.Context, *tickets.Ticket) error`. El método
`GenerateCoupon` incrementa atómicamente el contador global existente y
formatea el valor devuelto con padding mínimo de cuatro posiciones. El
registro inicial debe prepararse una sola vez en el entorno; el método no lo
recrea ni reinicia si falta o se agota.

`Save`, en el mismo adaptador `tickets/adapters/dynamodb`, guarda el tiquete
canónico y reserva el cupón mediante un único
`TransactWriteItems`, con condiciones de no existencia para ambos ítems.
Los duplicados y demás errores se propagan; no hay escrituras alternativas
tras una cancelación. El mapeo convierte los números de cada apuesta a una
lista numérica sin modificar la entidad.

El adaptador conserva `CreatedAt` como `time.Time` en el dominio y persiste
`createdAt` como String en UTC con milisegundos de ancho fijo. El mismo ítem
incluye las claves del listado por TIR en `GSI1` y por nodo y fecha en `GSI2`,
cuyo sort key es `GSI2SK`, con el mismo valor que `createdAt`. El despliegue de
esos índices pertenece al repositorio central de infraestructura. Consulta el
[modelo de tiquetes](data-model/ticket-access-patterns.md#índices-de-consulta) y el
[ADR 0027](adrs/0027-ordenar-gsi2-por-gsi2sk-con-fecha-iso.md).

El repositorio recibe el cliente DynamoDB compartido y `config.Config` mediante
`NewDynamodbRepository`. `ticketsfx.Module` lo registra y conecta al handler
HTTP mediante el caso de uso. Las operaciones financieras siguen pendientes.

## Infraestructura de suscripciones

`cmd/script` lee `Name()` y `SubscribeTo()` de los suscriptores registrados por
`nodesfx.WorkerModule` y `usersfx.WorkerModule`. Obtiene los patrones mediante `Type()` de cada prototipo,
conservando nombres exactos y prefijos con `*` final. Extrae colas, reglas y destinos, y usa plantillas Go
para escribir archivos Terraform en `infra/eventbridge/generated`. Cada
ejecución elimina esa carpeta y la regenera. Conserva el arranque Fx y sus
dependencias, sin ejecutar manejadores ni crear recursos con el SDK.
`infra/eventbridge` consume la salida como módulo `subscriptions`, configura
el provider y recibe región, ambiente y `event_bus_name` para un bus existente.
El estado es local por defecto; el backend, la configuración manual y los
valores del despliegue viven fuera de `generated/`. Solo se ejecuta Terraform
después de una generación exitosa. Consulta el
[ADR 0010](adrs/0010-administrar-eventos-desde-infra-eventbridge.md).
