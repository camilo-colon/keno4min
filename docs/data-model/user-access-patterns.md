# Patrones de acceso de usuarios

Los usuarios recibidos mediante `cronos.core.user.created` se replican en la
tabla configurada por `DYNAMODB_TABLE`, con un ítem por ID de usuario.

| Atributo | Tipo DynamoDB | Contenido |
| --- | --- | --- |
| `PK` | String | `USER#<id>` |
| `SK` | String | `META` |
| `GSI1PK` | String | `USER#<username>` |
| `GSI1SK` | String | `META` |
| `id` | String | UUID del usuario (`detail.userId`) |
| `username` | String | `detail.username`, conservando mayúsculas y minúsculas |
| `roleId` | Number | `detail.roleId`, incluido el valor cero |
| `nodeId` | String | `detail.nodeId`, UUID con guiones |

La identidad es `User.ID`, recibido como `UserCreatedDomainEvent.UserID`
(`detail.userId`). No se normaliza el username; se utiliza como clave del índice,
manteniendo el ID como clave principal.
El ID del evento (`detail.eventId`) se mantiene separado: no se almacena ni se
utiliza como identidad del usuario. Se reutiliza el índice compartido `GSI1`.

| ID | Operación | Acceso | Condición |
| --- | --- | --- | --- |
| `AP-USER-01` | Crear un usuario si no existe | `PutItem` en `USER#<id>` / `META` | `attribute_not_exists(PK)` |
| `AP-USER-02` | Obtener un usuario por ID | `GetItem` en `USER#<id>` / `META`, con lectura fuerte | No aplica |
| `AP-USER-03` | Obtener un usuario por username | `Query` en `GSI1`, con `GSI1PK = USER#<username>` y `GSI1SK = META` | Consistencia eventual |

`FindById` devuelve `(nil, nil)` cuando no existe el ítem. Propaga errores
de lectura o conversión de atributos y rechaza ítems sin username o con un
id o nodeId que no pueda convertirse a UUID.

`FindByUsername` devuelve `(nil, nil)` si no hay coincidencias y conserva la
distinción entre mayúsculas y minúsculas. Ejecuta un único `Query` con
`Limit: 1` y devuelve la primera coincidencia, sin paginación ni comprobación
de duplicados.
Propaga errores de DynamoDB y de conversión con el username consultado.

El índice no garantiza unicidad ni visibilidad inmediata después de una
escritura. No debe usarse para comprobar unicidad al crear usuarios; la
creación sigue siendo idempotente por ID. Backoffice administra los usernames.

`GSI1` debe usar `GSI1PK` y `GSI1SK` de tipo String y proyectar `id`, `username`,
`roleId` y `nodeId`, mediante `INCLUDE` o `ALL`. La consulta convierte los datos
proyectados directamente al usuario. La configuración del índice se administra
fuera de este repositorio. El rol que ejecute esta consulta necesita
`dynamodb:Query` sobre el ARN de `index/GSI1` de la tabla.

Los nuevos usuarios se guardan con ambas claves del índice. Los usuarios
existentes sin esos atributos requieren un backfill para aparecer en la
búsqueda: repetir el evento no los añade porque la creación conserva el ítem
existente. Este cambio no ejecuta un backfill ni modifica recursos AWS.

`users/usecase.Create` rechaza usernames vacíos o compuestos solo por espacios
y userId o nodeId ausentes o iguales al UUID cero. Primero consulta el usuario
por ID; si ya existe, termina con éxito. Si no existe, guarda los datos recibidos. No consulta
el nodo asociado, de modo que los eventos de usuarios y nodos pueden llegar
en cualquier orden.

`Save` utiliza una escritura condicional para proteger también las creaciones
concurrentes. Si otro proceso ya creó el usuario, devuelve éxito conservando
todos los atributos existentes, incluso cuando la reentrega contiene datos
distintos. Esta operación no actualiza roles, asociaciones ni usernames.

Los demás errores llegan al consumidor, que conserva el mensaje en SQS para
reentrega o DLQ. El worker requiere `dynamodb:GetItem` y `dynamodb:PutItem`
sobre la tabla. La generación de infraestructura incorpora el suscriptor
`cronos.keno4min.user.create_on_user_created`; los cambios se despliegan mediante
el flujo Terraform existente.

El formato anterior `USER#<username>` no se consulta ni se migra automáticamente.
Si se almacenaron usuarios con ese formato, deben migrarse antes de usar esta
versión. El username ya no impone unicidad en esta réplica.
