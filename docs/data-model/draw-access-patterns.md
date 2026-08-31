# Patrones de acceso de draws

## Ítems

| Ítem | PK | SK | Contenido |
| --- | --- | --- | --- |
| Draw | `DRAW#<drawId>` | `META` | Fuente de verdad |
| TIR lookup | `DRAW_TIR#<tir>` | `LOOKUP` | `drawId` |
| Estado de lectura | `DRAW_STATE` | `META` | `current_draw` y `last_drawn` |
| Historial | `DRAW_DRAWN` | `TIR#<tir>` | Proyección final del draw |

## Patrones de acceso

| Patrón | Tabla/índice | Operación | Valor de PK | Valor de SK |
| --- | --- | --- | --- | --- |
| `getCurrentDraw` | Tabla base | `GetItem` | `DRAW_STATE` | `META` |
| `getLastDrawn` | Tabla base | `GetItem` | `DRAW_STATE` | `META` |
| `listDrawnHistory(beforeTir)` | Tabla base | `Query`, descendente y paginado | `DRAW_DRAWN` | `< TIR#<beforeTir>` |
| `getDrawById` | Tabla base | `GetItem` | `DRAW#<drawId>` | `META` |
| `getDrawByTir` | Tabla base | Dos `GetItem` | `DRAW_TIR#<tir>` → `DRAW#<drawId>` | `LOOKUP` → `META` |

Todas las lecturas usan consistencia fuerte. El historial utiliza
`ScanIndexForward=false` y `LastEvaluatedKey`; `beforeTir` excluye el resultado
que ya se muestra como `last_drawn`. En `DRAW_DRAWN`, `tir` se completa con ceros
a la izquierda para conservar el orden numérico. Step Functions actualiza
`current_draw` en `create_draw` y `last_drawn` junto con el historial en `draw`,
de forma transaccional.
