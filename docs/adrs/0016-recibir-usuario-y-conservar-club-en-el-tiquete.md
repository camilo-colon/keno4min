# 0016. Recibir el usuario y conservar el club en el tiquete

- Estado: Reemplazada
- Fecha: 2026-10-20
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0015](0015-recibir-identificador-del-tiquete-desde-el-frontend.md)
- Reemplazada por: [ADR 0017](0017-conservar-vendedor-como-created-by.md)

## Contexto

La creación se está definiendo por etapas. Se acuerda recibir el usuario
vendedor directamente y obtener el club de su `NodeId`, aplazando las
comprobaciones de autenticación, permisos, club y admisión del sorteo. El
tiquete debe conservar la asociación de la venta aunque el usuario cambie de
nodo posteriormente.

## Decisión

Usar `GET /tickets` con los parámetros obligatorios `id`, `userId`, `stv` y
`sum`, sin body ni ID como segmento de ruta. El frontend genera el UUID v7 del
tiquete. El servidor consulta el usuario por `userId` y guarda su identidad y
`user.NodeId` como `UserID` y `NodeID` del tiquete, respectivamente. En este
flujo `NodeID` representa al club y no se consulta el nodo para determinar esa
asociación.

Se mantienen el contador global atómico, el formato de cupón de 16 dígitos,
los saltos permitidos, la única región de asignación y la reserva permanente
guardada junto al tiquete en una transacción con condiciones de no existencia.

El flujo actual valida la entrada, la existencia del usuario y las apuestas,
calcula el total y conserva la comprobación de saldo como paso pendiente de
concretar. La autenticación y permisos del vendedor, las comprobaciones del
club y la validación del estado y horario del sorteo se aplazan. La
idempotencia continúa pendiente.

La estructura de creación queda definida en el modelo de tiquetes, sin
`RulesVersion`, porque la tabla de pagos es fija. La asignación de `DrawID` y
la fuente de `Currency` siguen pendientes.

## Alternativas consideradas

- Identificar al vendedor mediante autenticación y consultar adicionalmente
  su pertenencia al club: se aplaza para la etapa actual.
- Resolver el club desde el usuario cada vez que se consulta el tiquete: no
  preserva necesariamente el club al que pertenecía la venta original.
- Versionar la tabla de pagos dentro del tiquete: se omite mientras exista una
  única tabla fija; un cambio futuro deberá preservar los tiquetes anteriores.

## Consecuencias

- Un usuario inexistente impide crear el tiquete. Recibir su UUID y consultarlo
  no autentica al vendedor.
- La asociación al club queda conservada con el tiquete.
- Las comprobaciones aplazadas y las fuentes pendientes permanecen explícitas
  en el PRD; esta decisión no declara completo el flujo de venta del MVP.
- Se documenta el dominio y la persistencia previstos; no se implementan
  endpoints ni paquetes en este cambio.

## Referencias

- [PRD: flujo actual](../product-requirements.md#733-flujo-de-creación-de-la-etapa-actual).
- [Dominio y persistencia del tiquete](../data-model/ticket-access-patterns.md).
- [Consulta de usuarios](../data-model/user-access-patterns.md).
