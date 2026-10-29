# 0024. Tomar el vendedor del token al crear tiquetes

- Estado: Aceptada
- Fecha: 2026-10-28
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0019](0019-generar-identificador-del-tiquete-en-el-backend.md)

## Contexto

El middleware `platform/auth` se monta en todas las rutas y expone una
`auth.Identity` decodificada del JWT validado por el API Gateway
([ADR 0023](0023-delegar-la-validacion-del-token-al-api-gateway.md)). La
creación de tiquetes seguía recibiendo `userId` en la query, de modo que el
cliente podía indicar un vendedor distinto del autenticado.

## Decisión

La creación de tiquetes recibe únicamente `stv` y `sum`. El handler obtiene el
UUID del vendedor de `Identity.UserID` (claim `sub`) mediante
`auth.FromContext` y lo entrega al caso de uso en `CreateParams.UserID`. Una
query que incluya `userId` se rechaza como parámetro no permitido. Si la
identidad no está disponible, el handler responde `401`.

Se mantienen las demás decisiones del ADR 0019: el backend genera el UUID v7
del tiquete; el caso de uso consulta al usuario por ID, copia `user.Username`
a `CreatedBy` y `user.NodeId` a `NodeID`, y no persiste el UUID del usuario.

## Alternativas consideradas

- Tomar también `Username` y `NodeID` de los claims del token: evita la lectura
  del usuario, pero dejaría de comprobar que el vendedor existe en Keno4min y
  cambiaría la fuente del club; se aplaza.
- Aceptar `userId` y compararlo con el token: mantiene el parámetro sin aportar
  información y añade un caso de error innecesario.

## Consecuencias

- El cliente deja de enviar `userId`; las integraciones que lo envíen reciben
  `400` hasta retirarlo.
- El vendedor ya no puede suplantarse desde la query. La confianza en la
  identidad depende de que la API solo sea accesible a través del API Gateway.
- El caso de uso no cambia ni depende de `platform/auth`.
- Las comprobaciones de permisos, actividad y pertenencia del club siguen
  aplazadas.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Arquitectura](../architecture.md).
