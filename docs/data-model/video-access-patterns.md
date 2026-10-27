# Importación temporal de videos

`tmp_migrate_videos.py` lee MongoDB y crea ítems en `cronos-dev-keno4min`
(tabla predeterminada). Selecciona `game = keno4min` (configurable con `--game`), sin filtrar por `status`,
y utiliza únicamente `idv` como dato de origen.

| Atributo | Tipo DynamoDB | Contenido |
| --- | --- | --- |
| `PK` | String | `VIDEO` |
| `SK` | String | `idv` completo, incluyendo consecutivo y balotas |
| `id` | String | El mismo `idv`, sin modificar |
| `balls` | List de Number | Balotas posteriores al primer `_`, separadas por comas y convertidas a enteros |

Se conserva el orden de las balotas; `03` se almacena como el número `3`.
El consecutivo permanece en `SK` e `id`, sin un atributo adicional.
Se requieren 20 balotas distintas entre 1 y 80. Los documentos inválidos se
omiten, se reportan por `_id` y causan un código de salida distinto de cero.

La escritura es `PutItem` sobre `VIDEO / <idv>`, con la condición
`attribute_not_exists(PK)`. Un ítem existente se conserva y se cuenta como
omitido. Se puede repetir la importación después de un fallo; los demás errores
de MongoDB o AWS detienen el proceso y se imprime el resumen parcial.
El rol AWS requiere `dynamodb:PutItem` sobre la tabla de destino.

Por defecto se simula: se leen y validan los documentos, se muestran hasta
tres ítems de ejemplo y no se conecta a AWS. `--write` habilita la escritura;
`--limit N` limita los documentos de origen leídos. La simulación no comprueba
si los ítems ya existen en DynamoDB.

Si la consulta no devuelve documentos, lee únicamente el campo `game` de un
documento de muestra para distinguir una colección vacía de un filtro sin
coincidencias. Informa la base y colección consultadas y el filtro exacto;
no muestra la URI de conexión.

Instalar dependencias con `python3 -m pip install boto3 pymongo`. Configurar
`MONGODB_URI`, `MONGODB_DATABASE` y `MONGODB_COLLECTION` en el entorno; la base
y colección también aceptan `--database` y `--collection`. No se carga `.env`
automáticamente. Las credenciales AWS se resuelven mediante el SDK, incluido
`AWS_PROFILE` o `--profile`. La región se toma de `--region`, `AWS_REGION`,
`AWS_DEFAULT_REGION` o, si faltan, `us-east-1`.
