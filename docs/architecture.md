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
├── platform/workerfx/       # Consumidores SQS, deserialización y ciclo de vida Fx
├── users/
│   ├── user.go              # Réplica del usuario de backoffice
│   ├── events/              # Contrato UserCreatedDomainEvent
│   ├── ports/               # Contrato de persistencia de usuarios
│   ├── usecase/             # Creación idempotente por ID
│   ├── subscribers/         # Adaptación del evento al caso de uso
│   ├── adapters/dynamodb/   # Lectura y escritura condicional
│   └── usersfx/             # Registro Fx para worker y generador
└── draw/
    ├── draw.go              # Entidad y reglas de negocio
    ├── get_by_id.go         # Caso de uso y contrato IDReader
    ├── dynamodb/
    │   ├── store.go         # Lectura del draw canónico mediante GetItem
    │   └── item.go          # Formato almacenado y decodificación
    └── http/
        └── handler.go      # Ruta por ID, respuesta JSON y errores HTTP
```

Las pruebas viven junto al código que verifican. Los paquetes y archivos se
crean cuando se implementa una responsabilidad, sin anticipar operaciones.

## Dependencias

- `draw` contiene la entidad, sus invariantes, los casos de uso y los contratos
  pequeños que estos consumen. No importa Fiber, AWS ni sus adaptadores.
- `draw/http` depende de `draw` y Fiber. Adapta parámetros, configura el plazo de
  la petición y construye respuestas HTTP, incluidos nombres JSON, UTC y bolas
  como array numérico.
- `draw/dynamodb` depende de `draw` y el SDK AWS. Construye claves, lee y convierte
  los datos, comprobando el formato de almacenamiento. Puede compartir esta
  implementación entre operaciones del módulo.
- Los adaptadores no se importan entre sí. Un adaptador puede implementar varios
  contratos pequeños; no se introduce un repositorio CRUD genérico.
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

La consulta por TIR anterior fue retirada. Las consultas por TIR, histórico y
runtime conservan su diseño de acceso documentado, pero no tienen implementación.
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

## Creación de tiquetes prevista

La creación de tiquetes todavía no está implementada. El contrato HTTP y las
reglas acordadas están en el [PRD, sección 7.3](product-requirements.md#73-creación).
El [modelo de tiquetes](data-model/ticket-access-patterns.md) define el dominio
`Ticket` y `Bet`, sus atributos persistidos, el contador global y la reserva
exclusiva del cupón. El [ADR 0017](adrs/0017-conservar-vendedor-como-created-by.md)
registra la decisión vigente: el frontend envía el UUID y `userId`; la creación
consulta al usuario y conserva `user.NodeId` como club de la venta y
`user.Username` en `CreatedBy` como nombre histórico del vendedor. El tiquete
no conserva el UUID del usuario; `userId` se usa para consultarlo durante la
creación. El listado incluirá `createdBy` y TIR desde los datos del tiquete,
evitando consultas por cada fila.

El [flujo de la etapa actual](product-requirements.md#733-flujo-de-creación-de-la-etapa-actual)
aplaza la autenticación y permisos, las comprobaciones del club, la validación
del estado y horario del sorteo y la idempotencia. La comprobación y descuento
de saldo y la resolución del draw para obtener `DrawID` y `TIR` siguen pendientes.
Esta previsión no cambia las rutas implementadas ni introduce paquetes nuevos.

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
