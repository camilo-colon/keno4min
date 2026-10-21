# Patrones de acceso de nodos

Los nodos recibidos mediante `NodeCreatedDomainEvent` se almacenan en la tabla
configurada por `DYNAMODB_TABLE`. Cada nodo ocupa un único ítem.

## Ítem

| Atributo | Tipo DynamoDB | Contenido |
| --- | --- | --- |
| `PK` | String | `NODE#<id>` |
| `SK` | String | `META` |
| `GSI1PK` | String | `NODE#<parentId>`; ausente en nodos sin padre |
| `GSI1SK` | String | `NODE#<id>`; ausente en nodos sin padre |
| `id` | String | UUID del nodo (`detail.nodeId`) |
| `parentId` | String | UUID del padre; ausente cuando `ParentId` es `nil` |
| `type` | String | Tipo de nodo (`Admin` o `Cashier`) |
| `name`, `address`, `timeZone`, `currency` | String | Valores recibidos del nodo |
| `balanceCents` | Number | Saldo entero en centavos recibido al crear el nodo |
| `active` | Boolean | Estado recibido al crear el nodo |

Los UUID se guardan como texto con guiones. El saldo cero y `active=false`
se almacenan explícitamente. Al guardar nodos con padre se incluyen las claves
de `GSI1`; los nodos sin padre omiten ambas claves. No se añaden ítems auxiliares.

## Patrones implementados

| ID | Operación | Acceso | Condición |
| --- | --- | --- | --- |
| `AP-NODE-01` | Crear un nodo si no existe | `PutItem` en `NODE#<id>` / `META` | `attribute_not_exists(PK)` |
| `AP-NODE-02` | Obtener un nodo por ID | `GetItem` en `NODE#<id>` / `META`, con lectura fuerte | No aplica |

`FindById` consulta la tabla base con `ConsistentRead=true` y devuelve `(nil, nil)`
si el nodo no existe. Convierte los atributos al dominio, preservando `ParentId=nil`
cuando no hay padre, y propaga los errores de DynamoDB, deserialización o UUID
inválidos. El rol que realiza esta lectura necesita `dynamodb:GetItem` sobre la tabla.

`Save` realiza una única escritura condicional, sin lectura previa. Si el ítem
existe, devuelve éxito y conserva todos sus atributos, incluso si el evento
recibido contiene valores diferentes. Esto permite reentregar una creación sin
sobrescribir cambios posteriores. La identidad usada es la del nodo, no la del
evento. `Save` no sirve para actualizar nodos ni sincronizar sus cambios.

Los demás errores se propagan al consumidor, que no elimina el mensaje de SQS.
El rol del worker necesita `dynamodb:PutItem` sobre la tabla.

El caso de uso `nodes/usecase.Create` combina estos accesos: primero consulta
el ID del nodo y termina con éxito si ya existe, sin revalidar el padre ni
sobrescribirlo. Para nodos nuevos, consulta el padre cuando `ParentId != nil`
y solo llama a `Save` si lo encuentra; un nodo raíz omite la consulta al padre.
Un padre ausente produce `ErrParentNotFound` y deja el mensaje disponible para
reintento, por ejemplo cuando los eventos llegan fuera de orden. Un padre que
nunca llegue requiere el tratamiento de la DLQ configurada.
El worker necesita tanto `dynamodb:GetItem` como `dynamodb:PutItem`.
La lectura del padre y la escritura del hijo no son una transacción: esta
comprobación no impide que otro proceso elimine al padre entre ambas operaciones.
La condición de `Save` sigue protegiendo contra creaciones concurrentes del
mismo nodo.

Las claves guardadas preparan el acceso por padre mediante `GSI1`, con
`GSI1PK` y `GSI1SK` de tipo String. La consulta de hijos no está implementada.
La configuración del índice se administra fuera de este repositorio.
Los nodos existentes no reciben estas claves al repetir `Save`, porque la
escritura conserva los registros existentes.

El formato de almacenamiento y su conversión viven en el adaptador; la entidad
`nodes.Node` no tiene etiquetas ni dependencias de DynamoDB. Consulta el
[ADR 0013](../adrs/0013-crear-nodos-solo-si-no-existen.md).
