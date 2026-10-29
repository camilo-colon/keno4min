# 0023. Delegar la validación del token al API Gateway

- Estado: Aceptada
- Fecha: 2026-10-28
- Responsables: Equipo de cronos

## Contexto

Los clientes envían un JWT emitido por cronos-core en el encabezado
`Authorization: Bearer`. Sus claims `sub`, `unique_name`, `role` y `node_id`
identifican al usuario, su rol y su nodo. El API Gateway valida la firma, el
emisor, la audiencia y la vigencia del token antes de enrutar la solicitud a
Keno4min.

## Decisión

Keno4min confía en el API Gateway como frontera de autenticación. El middleware
`platform/auth` solo decodifica el payload del token, sin verificar su firma ni
sus fechas, y guarda una `auth.Identity` que los handlers leen con
`auth.FromContext`. Responde `401` cuando el encabezado falta o el token no
puede leerse o carece de los claims requeridos.

El middleware no se monta en ninguna ruta en este cambio. Su uso en los
endpoints y el reemplazo de `userId` en la creación de tiquetes se decidirán
posteriormente.

## Alternativas consideradas

- Verificar también el token en Keno4min: añade defensa en profundidad, pero
  duplica la validación del gateway y obliga a distribuir el secreto de firma de
  cronos-core a este servicio.

## Consecuencias

- Keno4min no depende del secreto de firma ni de una librería JWT.
- La API no debe ser accesible sin pasar por el API Gateway. Si se expone
  directamente, cualquier cliente podría suplantar a otro usuario, rol o nodo.
- Si esa condición cambia, este ADR deberá reemplazarse por uno que reintroduzca
  la verificación del token.

## Referencias

- [Arquitectura](../architecture.md).
- [ADR 0017: vendedor como `createdBy`](0017-conservar-vendedor-como-created-by.md).
