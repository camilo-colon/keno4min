# 0020. Conectar la creación HTTP de tiquetes

- Estado: Reemplazada
- Fecha: 2026-10-24
- Responsables: Equipo de cronos
- Reemplazada por: [ADR 0021](0021-validar-ventana-del-sorteo-al-crear-tiquetes.md)

## Contexto

El contrato `GET /tickets` y el guardado transaccional existen, pero el handler
no está conectado al caso de uso y falta resolver el TIR. El registro HTTP
compartido antepone `/v1`, incompatible con la ruta acordada. La aceptación
financiera y la validación condicional del sorteo siguen fuera de esta etapa.

## Decisión

Componer la creación en `tickets/ticketsfx`, siguiendo los módulos de usuarios
y sorteos. Compartir el proveedor de usuarios mediante
`usersfx.RepositoryModule`, consumido por la API y por el worker.

Agregar el grupo Fx `rootRoutes` al registro del servidor para montar
`/tickets` en la raíz; conservar `routes` bajo `/v1` para las rutas existentes.
El adaptador HTTP interpreta y valida la query, reutiliza `tickets.ValidateBets`
y devuelve el tiquete en JSON solo después del guardado.

El caso de uso depende de `draws/ports.Repository` para leer `GetRuntime` y
tomar `Current.Tir` después de consultar al usuario y antes del contador. Deja
de recibir TIR como parámetro. No importa adaptadores ni usa el histórico.
La lectura fuerte del runtime solo determina la asociación por TIR: no
autoriza una venta frente al estado y horario del draw canónico.

## Consecuencias

- La API puede crear el registro del tiquete en estado `pending` y reservar
  su cupón en una misma transacción. El JSON y los códigos HTTP se definen en
  el PRD, sección 7.3.8.
- El runtime y el usuario deben existir, y el contador debe estar preparado.
  Un fallo de resolución del TIR impide consumir un cupón o guardar.
- Los límites técnicos de apuestas y query se configuran por entorno.
- `HEAD` no ejecuta la creación y las respuestas usan `no-store`.
- No se resuelven el descuento de saldo, los permisos, la idempotencia ni la
  validación condicional del sorteo. Guardar el ticket no confirma un débito.

## Referencias

- [ADR 0019: UUID en el backend](0019-generar-identificador-del-tiquete-en-el-backend.md).
- [Arquitectura](../architecture.md#creación-de-tiquetes).
- [PRD: creación](../product-requirements.md#73-creación).
- [Modelo de acceso de tiquetes](../data-model/ticket-access-patterns.md).
