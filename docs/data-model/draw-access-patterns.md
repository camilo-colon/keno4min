# Patrones de acceso de draws

Este modelo mantiene el draw completo como fuente de verdad, agrega un ítem para
resolver el TIR, usa un GSI sparse para el histórico y materializa el estado de
presentación del juego. No requiere operaciones `Scan`.

## Ítems

| Ítem | PK | SK | GSI1PK | GSI1SK | Contenido |
| --- | --- | --- | --- | --- | --- |
| Draw no sorteado | `DRAW#<drawId>` | `META` | No existe | No existe | Draw completo; es la fuente de verdad |
| Draw sorteado | `DRAW#<drawId>` | `META` | `DRAW#DRAWN` | `TIR#<tir20>` | Draw completo; es la fuente de verdad |
| Búsqueda por TIR | `DRAW_TIR#<tir>` | `META` | No existe | No existe | Solo `drawId` |
| Estado de presentación | `DRAW_RUNTIME` | `META` | No existe | No existe | `current` y `latestResult` |

`tir20` es el TIR decimal completado a 20 dígitos con ceros a la izquierda. Por
ejemplo, el TIR `3573` se almacena como `TIR#00000000000000003573`. Esta
normalización conserva el orden numérico cuando DynamoDB ordena la sort key como
texto.

`GSI1` proyecta, como mínimo, `drawId`, `tir`, `startsAt`, `closesAt`, `balls` y
`videoId`. Los atributos `GSI1PK` y `GSI1SK` se agregan al ítem canónico solamente
cuando el draw alcanza el estado `drawn`. Por ello, el índice es sparse y los
draws abiertos o cerrados sin resultado no pertenecen al histórico.

`DRAW_RUNTIME` es una proyección pequeña orientada a la pantalla del juego:

| Atributo | Contenido |
| --- | --- |
| `current` | `drawId`, `tir`, `startsAt` y `closesAt` del último draw publicado por `create_draw` |
| `latestResult` | `drawId`, `tir`, `balls` y `videoId` del resultado definitivo más reciente |

`current` y `latestResult` representan funciones distintas y pueden apuntar al
mismo draw. Esto sucede después de `draw` y hasta el siguiente `create_draw`.

## Patrones de acceso

`AP-DRAW-01` está implementado en la API. `AP-DRAW-02`, `AP-DRAW-03` y
`AP-DRAW-04` describen el diseño previsto; no tienen endpoints implementados.

| ID | Patrón | Destino | Operación | Clave o condición | Orden | Consistencia |
| --- | --- | --- | --- | --- | --- | --- |
| `AP-DRAW-01` | Obtener un draw por ID | Tabla base | `GetItem` | `PK = DRAW#<drawId>` y `SK = META` | No aplica | Fuerte |
| `AP-DRAW-02` | Obtener un draw por TIR | Tabla base | Dos `GetItem` | `DRAW_TIR#<tir>`/`META` para obtener `drawId`; luego `DRAW#<drawId>`/`META` | No aplica | Fuerte en ambas lecturas |
| `AP-DRAW-03` | Listar el histórico de draws sorteados | `GSI1` | `Query` paginado | `GSI1PK = DRAW#DRAWN` | TIR descendente | Eventual |
| `AP-DRAW-04` | Obtener el estado de presentación del juego | Tabla base | `GetItem` | `PK = DRAW_RUNTIME` y `SK = META` | No aplica | Fuerte |

`AP-DRAW-03` usa `ScanIndexForward = false`, un `Limit` definido por la API y
la consistencia eventual propia del GSI. Si DynamoDB retorna `LastEvaluatedKey`,
el servicio lo convierte en un cursor opaco y lo usa como `ExclusiveStartKey` en
la página siguiente. El índice proyecta los datos que necesita la lista, por lo
que el patrón no hace una lectura adicional por cada draw.

El último resultado puede tardar brevemente en aparecer en `AP-DRAW-03`. Los
flujos que necesitan observarlo inmediatamente deben usar `AP-DRAW-01`,
`AP-DRAW-02` o la proyección fuerte de `AP-DRAW-04`.

`AP-DRAW-04` evita consultar "los últimos dos draws". Durante la ventana de
apuestas, `current` identifica el draw que acepta tiquetes y `latestResult`
contiene el resultado anterior. Después de `draw`, ambos contienen el mismo TIR
y la pantalla puede presentar la animación del resultado actual. El siguiente
`create_draw` reemplaza únicamente `current` y vuelve a separarlos.

Esta proyección es solo para lectura y presentación. Una operación de negocio
debe validar condicionalmente el estado y la ventana temporal sobre el draw
canónico; `DRAW_RUNTIME` no autoriza la creación, repetición ni anulación de
tiquetes.

## Relación con el flujo del sorteo

```text
create_draw -> wait -> draw -> update_tickets
     |                    |
     |                    ├─ finaliza el draw y agrega las claves de GSI1
     |                    └─ publica el resultado en latestResult
     └─ crea el draw y la búsqueda por TIR; reemplaza current
```

`wait` no modifica el modelo y `update_tickets` no controla la incorporación al
histórico. Un draw empieza a replicarse hacia `GSI1` cuando la etapa `draw`
persiste su resultado definitivo.

## Reglas de escritura

- La creación escribe el draw canónico, la búsqueda por TIR y
  `DRAW_RUNTIME.current` mediante un único `TransactWriteItems`. Los dos primeros
  usan condiciones de no existencia para impedir que un `drawId` o un TIR se
  asignen más de una vez. `latestResult` permanece intacto.
- La actualización de `current` exige que no exista o que su `tir` sea menor que
  el del draw candidato. Una ejecución tardía o repetida no puede hacer
  retroceder el ciclo visible.
- Al establecer el resultado, `draw` actualiza condicionalmente el ítem canónico,
  persiste `balls` y agrega `GSI1PK` y `GSI1SK` en la misma escritura. La
  condición impide reemplazar `balls` y hace que el procesamiento repetido
  converja en el resultado ya persistido.
- Después de confirmar el resultado canónico, `draw` actualiza
  `DRAW_RUNTIME.latestResult`. La condición exige que no exista o que su `tir`
  sea menor que el resultado candidato. Un resultado atrasado no puede
  reemplazar uno más reciente.
- La tarea `draw` solo termina cuando `latestResult` fue actualizado, ya contiene
  el mismo resultado o contiene un TIR posterior. El mismo TIR con bolas
  diferentes es un error de integridad. Si el proceso falla entre el resultado
  canónico y la proyección, el retry reutiliza las bolas persistidas y repara
  `DRAW_RUNTIME`; nunca genera otro resultado.
- Los ítems `DRAW_TIR#<tir>` son inmutables. Por eso, después de una creación
  confirmada, las dos lecturas fuertes de `AP-DRAW-02` resuelven siempre el mismo
  draw.
- El resultado y los atributos del índice no se modifican después de alcanzar
  `drawn`. El modelo presupone que el TIR es único y crece con cada draw; si esa
  regla de negocio cambia, la sort key del histórico deberá basarse en otro
  valor cronológico.

## Decisiones y límites

### Contrato implementado para la consulta por ID

`AP-DRAW-01` se expone como `GET /api/v1/draws/{id}`. El ID es opaco y no puede
estar vacío ni contener solo espacios en blanco. El cliente lo codifica como un
segmento URL; HTTP lo decodifica una vez y el caso de uso lo conserva exactamente,
sin recortar espacios ni cambiar mayúsculas.

`draw.GetByID` ejecuta la consulta a través de `draw.IDReader`. El adaptador
`internal/draw/dynamodb` hace un único `GetItem` con `PK = DRAW#<id>`, `SK = META`
y `ConsistentRead=true` en la tabla base. La API no escribe ni consulta un GSI o
un lookup por TIR para esta operación. El rol necesita `dynamodb:GetItem` sobre
la tabla; los permisos se administran en el repositorio de infraestructura.

La tarea que escribe los draws debe respetar estos atributos DynamoDB:

| Atributo | Tipo | Regla |
| --- | --- | --- |
| `PK`, `SK` | String | Claves definidas en la tabla de ítems |
| `drawId` | String | Obligatorio tanto en el lookup como en el draw canónico |
| `tir` | Number | Entero positivo uint64, obligatorio en el canónico |
| `status` | String | `open`, `closed` o `drawn` |
| `startsAt`, `closesAt` | String | RFC 3339; `closesAt > startsAt` |
| `balls` | List de Number | En `drawn`, 20 enteros distintos entre 1 y 80; en otros estados ausente o lista vacía |
| `videoId` | String | Opcional; ausencia representada como cadena vacía en HTTP |

`balls` es una lista, no un Number Set: se conserva el orden almacenado.
El adaptador DynamoDB decodifica el draw canónico con `attributevalue.UnmarshalMap`
del SDK oficial y un struct con etiquetas `dynamodbav`, y devuelve `draw.Draw`.
Comprueba los tipos String y List para evitar conversiones permisivas de timestamps
numéricos o conjuntos sin orden. Los atributos adicionales se ignoran.

El caso de uso invoca `Draw.Validate()` para validar identidad, tiempos, estado y
bolas con las reglas compartidas del dominio, y comprueba que el ID recuperado
coincida con el solicitado. La persistencia no valida reglas de negocio.
El adaptador HTTP transforma los nombres a `snake_case`, normaliza los tiempos a
UTC y devuelve `balls: []` antes de que exista un resultado. Las bolas siempre se
serializan como un array numérico. El contrato HTTP completo está en
[OpenAPI](../../api/openapi/v1/openapi.yaml).

Un ítem inexistente produce `404`. Datos mal formados, una entidad inválida o una
discrepancia de ID producen `500`, igual que los errores de infraestructura. Un
ID inválido en los datos almacenados nunca se clasifica como error de entrada.
Los detalles internos no se exponen al cliente. La lectura tiene un plazo máximo
de cinco segundos y las respuestas exitosas usan `Cache-Control: no-store` porque
el estado puede cambiar.

### Histórico

El histórico acepta consistencia eventual y no participa en la publicación
inmediata del resultado, por lo que un GSI evita mantener manualmente otro ítem
por draw. La PK única mantiene simple el orden global y la paginación. Si las
métricas muestran concentración de lecturas o *throttling* sobre
`DRAW#DRAWN`, se revisará el patrón para introducir particiones temporales o una
capa de caché; no se anticipa ese costo en el MVP.
