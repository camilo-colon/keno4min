# Keno4min — Product Requirements Document

| Campo | Valor |
| --- | --- |
| Estado | Borrador |
| Versión | 0.13 |
| Última actualización | 2026-10-21 |
| Audiencia | Producto, ingeniería y QA |

## 1. Resumen

Keno4min es un servicio de Keno independiente orientado inicialmente a puntos de venta. El producto administra el ciclo completo del juego: sorteos globales cada tres minutos, venta y liquidación de tiquetes, premios y un jackpot progresivo independiente por club.

Aunque el nombre del producto incluye «4min», la frecuencia real de los sorteos es de **tres minutos**.

El MVP será operado por cajeros. La venta directa a jugadores mediante cuentas y una aplicación web se contempla para una fase posterior.

Keno4min no representa ni implementa el Keno oficial de una jurisdicción concreta. Cualquier operación con dinero real deberá cumplir la legislación y los procesos de certificación aplicables en cada mercado.

## 2. Objetivo del producto

Proporcionar un motor de Keno centralizado, consistente y auditable que permita a una red jerárquica de operadores vender tiquetes desde sus clubs, ejecutar un resultado común para todos ellos, liquidar premios con exactitud y administrar jackpots independientes por club.

## 3. Usuarios y entidades del dominio

### 3.1 Administrador

Un administrador puede tener cualquier cantidad de subadministradores. Cada subadministrador puede, a su vez, tener sus propios subadministradores, sin un límite funcional de profundidad definido.

La jerarquía, autenticación, moneda, saldos y configuraciones administrativas pertenecen al microservicio de backoffice y están fuera del alcance de Keno4min.

### 3.2 Club

Un club es un punto de venta perteneciente a un administrador. Cada club:

- utiliza la moneda heredada de su administrador;
- contiene uno o más cajeros;
- mantiene su propio jackpot;
- obtiene su configuración de apuestas y jackpot desde backoffice.

### 3.3 Cajero

Un cajero pertenece exclusivamente a un club. En el MVP, el cajero:

- crea y repite tiquetes;
- consulta los tiquetes que vendió y sus estados;
- anula tiquetes dentro del periodo permitido;
- paga exclusivamente los tiquetes ganadores que él mismo vendió.

### 3.4 Jugador

En el MVP el jugador no interactúa directamente con Keno4min ni mantiene una cuenta dentro del flujo de juego. Solicita sus apuestas al cajero y conserva el tiquete para reclamar un premio.

Las cuentas de jugador y las apuestas directas desde la web quedan fuera del alcance del MVP.

## 4. Alcance del MVP

Keno4min es responsable de:

- programar y registrar sorteos globales cada tres minutos;
- extraer 20 números únicos de un universo del 1 al 80;
- aceptar tiquetes con múltiples apuestas;
- cerrar la venta y repetición de tiquetes cuando el contador del sorteo alcanza `closesAt`;
- anular tiquetes válidos antes del cierre;
- contar aciertos y calcular premios con la tabla fija del juego;
- conservar el historial y estado de cada tiquete;
- liquidar cada jackpot por club;
- seleccionar de forma uniforme el tiquete ganador de un jackpot;
- coordinar débitos, reintegros y créditos con backoffice;
- impedir operaciones financieras o pagos duplicados;
- exponer información suficiente para operar y auditar el juego.

Quedan fuera del alcance del MVP:

- cuentas y apuestas directas de jugadores;
- compra anticipada de varios sorteos con un mismo tiquete;
- administración de usuarios, jerarquías, permisos, monedas y saldos;
- compra o asignación jerárquica de saldo;
- interfaz de selección rápida; el cliente de escritorio del cajero genera los números y envía la selección final;
- configuración administrativa de clubs y jackpots.

## 5. Dependencia de backoffice

Backoffice es la fuente de verdad para:

- administradores, subadministradores, clubs y cajeros;
- pertenencia de un cajero a un club;
- moneda heredada por la jerarquía;
- saldos de administradores, clubs y cajeros;
- valores mínimo y máximo permitidos por apuesta en cada club;
- porcentaje de contribución al jackpot por club;
- monto semilla y límites mínimo y máximo del objetivo del jackpot por club.

El modelo comercial trata el saldo como inventario de venta. Al crear un tiquete, el total apostado se descuenta del saldo del cajero. Al anularlo, se reintegra. Cuando el cajero paga un premio en efectivo, el valor pagado se acredita a su saldo. Este crédito solo ocurre cuando el cajero confirma el pago.

Las integraciones financieras deben ser idempotentes: reintentar una solicitud no puede duplicar un débito, reintegro o crédito.

## 6. Reglas del sorteo

1. Existe un sorteo global cada tres minutos.
2. Todos los clubs participan en el mismo sorteo y comparten sus 20 números ganadores.
3. Cada sorteo extrae 20 números distintos, sin reemplazo, del conjunto inclusivo `1..80`.
4. Cada combinación válida de 20 números debe tener la misma probabilidad de ocurrir.
5. Cada sorteo tiene una ventana semiabierta `startsAt <= tiempo < closesAt`.
   `closesAt` es el instante en que el contador visible llega a cero y se cierra
   el proceso de venta; no se deriva restando tres segundos a otro tiempo.
6. Durante el cierre no se pueden crear, repetir ni anular tiquetes para ese sorteo.
7. Amazon EventBridge Scheduler inicia una ejecución de AWS Step Functions cada tres minutos.
8. La ejecución debe ser idempotente: el mismo sorteo no puede generar dos resultados ni liquidarse dos veces, aunque el evento sea entregado o reintentado más de una vez.

### 6.1 Orquestación del ciclo

Por el momento, una máquina de estados de AWS Step Functions coordina cada
sorteo mediante el siguiente flujo secuencial:

```text
Amazon EventBridge Scheduler (cada 180 s)
  └─> create_draw
        └─> wait_draw (hasta closesAt)
              └─> draw (establecer balls)
                    └─> update_tickets
```

1. `create_draw` crea el sorteo correspondiente al intervalo programado y
   registra `startsAt` y `closesAt`. Para el flujo actual, `closesAt` se fija en
   `startsAt + 157 segundos`.
2. `wait_draw` suspende la ejecución hasta el timestamp absoluto almacenado en
   `closesAt`. Al alcanzarlo, la venta, repetición y anulación de tiquetes para
   ese sorteo ya están cerradas.
3. `draw` genera y persiste de forma definitiva los 20 valores de `balls` del
   sorteo, respetando las reglas de aleatoriedad de este documento.
4. `update_tickets` procesa los tiquetes válidos asociados al sorteo usando el
   resultado persistido y actualiza su liquidación.

Los 23 segundos entre `closesAt` y el siguiente intervalo programado constituyen
un margen operativo; no forman parte de la ventana de venta. Las entregas
duplicadas, los reintentos y la reanudación de una ejecución no pueden crear otro
sorteo para el mismo intervalo, reemplazar sus `balls` ni volver a liquidar sus
tiquetes.

## 7. Tiquetes y apuestas

### 7.1 Estructura conceptual

Cada tiquete:

- tiene un UUID v7 generado por el backend al crear el tiquete;
- registra el club mediante `NodeID`, copiado de `user.NodeId`;
- conserva `CreatedBy`, una copia de `user.Username` al crear la venta para mostrar al vendedor en el listado; no guarda `UserID` ni un campo separado `Username`;
- pertenece a un único sorteo, cuyo TIR se toma de `DRAW_RUNTIME.current.tir` durante la creación actual;
- se asocia al sorteo mediante su `TIR` global y único, que se muestra como número de tiro en el listado de tiquetes;
- contiene una o más apuestas;
- recibe un cupón numérico consecutivo con padding mínimo de 4 dígitos, único en toda la plataforma;
- conserva una copia inmutable de las apuestas aceptadas y sus valores;
- se liquida con la tabla fija de pagos; no incluye un campo `RulesVersion`.

La estructura de dominio acordada, sus tipos y su representación de
almacenamiento están en el [modelo de tiquetes](data-model/ticket-access-patterns.md#dominio-de-creación).
El tiquete no incluye un campo de moneda; esa configuración permanece en
backoffice. Sus importes siguen representándose como enteros en la unidad
monetaria mínima. El total se obtiene sumando los valores de sus apuestas y
no se guarda como un campo adicional del tiquete.

Ejemplo conceptual de apuestas; este JSON describe los datos y no es el body
del endpoint de creación:

```json
{
  "bets": [
    { "nums": [1, 2, 3], "value_cents": 50000 },
    { "nums": [10, 20, 30], "value_cents": 25000 }
  ]
}
```

El nombre `value_cents` representa la unidad monetaria mínima y se conserva por claridad aunque la moneda configurada no denomine esa unidad como «centavo».

### 7.2 Validación de una apuesta

Una apuesta es válida cuando:

- contiene entre 1 y 10 números;
- todos sus números son enteros entre 1 y 80;
- no repite un número dentro de la misma apuesta;
- su valor es un entero positivo expresado en la unidad mínima de la moneda;
- su valor respeta los límites configurados para el club;
- el cajero pertenece al club y dispone del saldo requerido;
- la solicitud se acepta antes del cierre del sorteo.

Estas reglas describen el objetivo funcional completo. La sección
[7.3.3](#733-flujo-de-creación-de-la-etapa-actual) distingue las comprobaciones
incluidas en la etapa actual de las que se aplazaron.

Un tiquete puede contener apuestas idénticas si el jugador así lo solicita. No existe un límite funcional de apuestas por tiquete en el MVP; el límite técnico configurable queda pendiente para una etapa posterior.

### 7.3 Creación

El total del tiquete es la suma de `value_cents` de todas sus apuestas. El tiquete solo queda aceptado si Keno4min puede garantizar el débito único de ese total en el saldo del cajero.

La coordinación financiera sigue pendiente de diseño. El flujo de esta etapa
no da por resuelto el débito por el hecho de guardar el tiquete y su cupón.

#### 7.3.1 Contrato de entrada acordado

La creación se expone mediante `GET /tickets`, con las apuestas en la query
de la URL y sin body. La ruta no lleva los prefijos `/v1/keno4min`, el segmento
`create` ni un identificador del tiquete como segmento de ruta. El backend
genera el UUID v7 del tiquete; el frontend envía solo las apuestas y sus
valores. El usuario vendedor se toma del token autenticado (claim `sub`).

```http
GET /tickets?stv=1%3B2%3B3_10%3B20%3B30&sum=50000_25000
```

| Parámetro obligatorio | Formato | Interpretación del ejemplo |
| --- | --- | --- |
| `stv` | Números separados por `;`; apuestas separadas por `_` | Apuestas `[1, 2, 3]` y `[10, 20, 30]` |
| `sum` | Valores enteros en la unidad monetaria mínima, separados por `_` | Valores `50000` y `25000`, respectivamente |

`%3B` es la codificación URL de `;`. Los valores de `sum` corresponden por
posición a las apuestas de `stv`; sus cantidades deben coincidir. Ambos
parámetros deben estar presentes y no vacíos, sin grupos ni elementos vacíos.
Los números y valores deben poder interpretarse como enteros positivos; no se
aceptan importes decimales. Se aplican las comprobaciones de esta etapa descritas
en la sección 7.3.3. La suma de los valores debe comprobarse sin desbordamiento.
Los parámetros obligatorios repetidos se rechazan para evitar interpretaciones
ambiguas de la misma URL.

El formato conserva la representación del legacy para los dispositivos de los
cajeros. El beneficio de rendimiento deberá medirse; no se considera demostrado
por el transporte en la URL. La operación debe ejecutarse por una acción
explícita de venta y sus respuestas deben usar `Cache-Control: no-store`.

El servidor consulta el usuario mediante el UUID del token, comprueba que
exista y copia `user.NodeId` como club de la venta y `user.Username` como nombre
del vendedor en `CreatedBy`. Ese nombre se obtiene de la misma consulta; el
frontend no lo envía como parámetro adicional. El UUID del usuario se usa para
consultarlo y no se guarda en el tiquete. No realiza comprobaciones de permisos, actividad
o pertenencia del club en esta etapa. Tampoco necesita consultar el nodo para
determinar esa asociación. El frontend no envía un `nodeId` independiente.

El UUID v7 del tiquete, el cupón y el total se asignan en el servidor.
El TIR se toma de `Current.Tir` mediante una lectura fuerte de `DRAW_RUNTIME`.
El tiquete conserva ese número como asociación al sorteo; el frontend
no lo envía y no se almacena un UUID adicional del draw en el tiquete.
Después de identificar el draw desde runtime, el servidor consulta el sorteo
canónico y usa `Draw.AcceptsTicketsAt` para comprobar `status = open` y
`startsAt <= ahora < closesAt` antes de generar el cupón.

#### 7.3.2 Identidad, cupón y reintentos

El UUID identifica internamente al tiquete. El cupón es la referencia visible
para localizarlo y pagar sus premios; se conserva y transmite como texto para
mantener los ceros iniciales.

El cupón se obtiene de un contador global creciente de DynamoDB, independiente
del club, cajero y sorteo. Empieza en `1`, se representa con ceros a la izquierda
y admite el intervalo `1` a `9999999999999999`, ambos inclusive, manteniendo el
límite máximo acordado del contador. El padding mínimo es de 4 posiciones:
`1` se presenta como `0001`, `9999` como `9999` y `10000` como `10000`. El
formato no recorta dígitos ni reinicia la secuencia al superar cuatro posiciones.
Se usa `uint64` para procesar el contador en Go. El formato no sustituye la
validación del límite: al agotarlo se rechaza una nueva asignación sin reiniciar
el contador ni reutilizar números.

Se aceptan saltos por fallos, reintentos o ventas que no llegan a completarse.
El contador mide asignaciones, no ventas aceptadas. Cada cupón se reserva de
forma exclusiva en la misma transacción que guarda el tiquete. Si el cupón ya
está reservado, la transacción no guarda el segundo tiquete ni sobrescribe la
reserva. La reserva es permanente, incluso tras una anulación.

La reserva garantiza la unicidad del cupón. El UUID se genera en el backend
para cada creación; no identifica reintentos de una misma venta. La clave de
idempotencia y el comportamiento de los reintentos no están definidos. Ese
diseño y el protocolo de coordinación financiera de la sección 14.1 siguen
pendientes. La respuesta HTTP implementada se describe en la sección 7.3.8.

El [modelo de acceso de tiquetes](data-model/ticket-access-patterns.md) define
las claves y operaciones. El endpoint está conectado al caso de uso y a
DynamoDB mediante los módulos Fx de la API.

#### 7.3.3 Flujo de creación de la etapa actual

1. **Validar la entrada.** Exigir `stv` y `sum` sin valores vacíos ni
   parámetros repetidos o adicionales. Tomar el UUID del usuario del token.
   Los límites técnicos de query y cantidad de apuestas quedan pendientes.
2. **Consultar el usuario.** Buscarlo por el UUID del token; si no existe, rechazar la
   creación. Copiar `user.Username` a `CreatedBy` y `user.NodeId` a `NodeID`.
   No guardar el UUID del usuario en el tiquete.
3. **Validar las apuestas y calcular el total.** Rechazar grupos o elementos
   vacíos y cantidades diferentes de selecciones y valores. Cada apuesta debe
   contener entre 1 y 10 números enteros de `1..80`, sin repetidos dentro de
   ella, y un valor entero positivo en la unidad monetaria mínima. Conservar
   apuestas completas idénticas por separado. Calcular el total con aritmética
   entera y detección de desbordamiento. Los límites monetarios por club quedan
   pendientes de su fuente de configuración.
4. **Comprobar el saldo.** Se mantiene como paso del flujo, pero su fuente
   autoritativa, el descuento y la coordinación financiera con backoffice
   siguen pendientes. Guardar el tiquete no confirma por sí solo un débito.
5. **Resolver el TIR y asignar el cupón.** Obtener el draw canónico mediante
   `draws.Repository.GetCurrent`, que encapsula las lecturas fuertes del runtime
   y del draw identificado por `Current.DrawId`. Usar su TIR y llamar a
   `Draw.AcceptsTicketsAt` con la hora UTC actual. Si no está abierto o está
   fuera de `startsAt <= ahora < closesAt`, rechazar con `409`, sin consumir
   cupón ni guardar. Después incrementar el contador global y formar el cupón
   según la sección 7.3.2.
6. **Construir y guardar.** Generar el UUID v7 del tiquete y usar el club obtenido
   del usuario, `CreatedBy` copiado de su username, las apuestas validadas,
   el total calculado y el cupón asignado.
   El servidor fija `CreatedAt` en UTC y el estado inicial `pending`. Guardar
   el tiquete y su reserva en una misma transacción, sin sobrescribir un UUID
   ni un cupón existentes.

La implementación actual del caso de uso se detalla en la sección 7.3.7:
reutiliza las reglas de apuestas del dominio desde el adaptador HTTP antes de
consultar repositorios. El constructor conserva sus comprobaciones después de
asignar el cupón; un rechazo posterior todavía puede dejar saltos permitidos.
La coordinación financiera sigue pendiente.

Los permisos del vendedor, las comprobaciones de actividad y
pertenencia del club y el diseño de idempotencia quedan fuera de esta etapa. Su aplazamiento no elimina
los requisitos funcionales finales del MVP.

El runtime identifica el draw; la admisión y el TIR se comprueban contra el
sorteo canónico. La validación es previa al cupón y al guardado, sin una
condición transaccional sobre el draw. Las comprobaciones del club siguen
pendientes.
Los datos de liquidación, jackpot, pago y anulación se definirán al abordar
esas operaciones.

#### 7.3.4 Constructor de dominio implementado

`tickets.Create` recibe el UUID del tiquete, cupón, UUID del club, TIR, apuestas
y nombre del vendedor, y devuelve el tiquete o un error. Valida UUID no nulos,
TIR positivo, vendedor no vacío ni compuesto solo por espacios, cupón no vacío
y las reglas de selección y valores positivos de las apuestas. Detecta
desbordamientos al sumar sus valores sin almacenar ese total. Conserva las
apuestas recibidas, mantiene apuestas completas idénticas y establece el
estado inicial `pending` y la hora UTC del servidor.

El constructor solo exige la presencia del cupón. `Repository.GenerateCoupon`
garantiza el rango y aplica el padding mínimo de 4 dígitos; el caso de uso lo
invoca antes de construir el tiquete. El caso de uso genera el UUID v7 del
tiquete; el parseo de la URL y del UUID del usuario corresponden a la entrada.
La existencia del usuario, los límites monetarios
por club, el saldo y las reservas en persistencia no se comprueban desde el
constructor. Las comprobaciones aplazadas de la sección 7.3.3 siguen pendientes.

#### 7.3.5 Guardado transaccional implementado

El puerto de persistencia expone `Save` y el adaptador DynamoDB guarda el
tiquete junto con la reserva permanente del cupón en una única transacción.
Ambas escrituras requieren que el ítem no exista. Un UUID existente o un cupón
reservado provoca un error sin sobrescribir los datos anteriores. El formato
de almacenamiento está en el [modelo de tiquetes](data-model/ticket-access-patterns.md#reserva-y-persistencia).

El adaptador recibe el tiquete con su cupón ya asignado; el caso de uso coordina
la generación del cupón, la construcción del tiquete y el guardado. El guardado
está conectado al handler HTTP, pero no confirma el débito de saldo.

#### 7.3.6 Generación del cupón implementada

El puerto `Repository` incluye `GenerateCoupon`, implementado en el adaptador
DynamoDB existente. El método realiza un único incremento atómico sobre el
contador global, utiliza el nuevo valor devuelto por DynamoDB y aplica el
padding mínimo de 4 posiciones. La condición de escritura impide incrementar
un contador ausente, negativo o agotado. Una respuesta inválida produce un
error sin devolver un cupón.

El contador debe inicializarse una sola vez con `value = 0`, como preparación
del entorno. El método no lo crea ni reinicia automáticamente. Su asignación
permanece separada del guardado transaccional del tiquete y la reserva; los
saltos y límites conservan las reglas de la sección 7.3.2.

#### 7.3.7 Caso de uso de creación implementado

`tickets/usecase.Create.Execute` recibe el UUID del usuario y las apuestas ya
interpretadas. Consulta `draws.Repository.GetCurrent` después de encontrar al
usuario y toma el TIR del draw canónico; el frontend no envía ese número.
El repositorio resuelve el UUID desde runtime y lee el draw. Un error de
lectura, un runtime/draw ausente o datos inválidos impiden generar el cupón y guardar.
Antes del contador, `Draw.AcceptsTicketsAt` verifica el estado y el horario.

Rechaza un UUID de usuario nulo y lo consulta por `FindById` usando el puerto
existente de usuarios. Si no existe, devuelve `ErrUserNotFound` sin generar
cupón ni guardar. Obtiene el cupón mediante `GenerateCoupon`, genera el UUID v7
con `uuid.NewV7()` y construye el tiquete con `tickets.Create`, copiando
`user.NodeId` a `NodeID` y `user.Username` a `CreatedBy`. El constructor mantiene las validaciones de
dominio. Si lo acepta, ejecuta `Save` y devuelve el tiquete guardado.

Todos los errores se propagan y detienen el flujo; un fallo de `Save` no
devuelve un tiquete como resultado exitoso. El adaptador interpreta las apuestas;
el constructor aplica `validateBets` y las comprobaciones restantes después
de generar el cupón.
Un rechazo posterior puede consumir un consecutivo sin guardarlo, conforme a
los saltos permitidos. No se duplican las reglas ni se reutiliza ese consecutivo.

`ticketsfx.Module` registra repositorio, caso de uso y handler. `cmd/api`
incorpora ese módulo y el repositorio compartido de usuarios. La ruta se
registra en la raíz; el runtime conserva `/v1/draws/runtime`. El saldo y las
comprobaciones aplazadas conservan el alcance pendiente ya definido.

#### 7.3.8 Respuesta HTTP y requisitos del entorno

El endpoint devuelve `201 Created` únicamente después de guardar el tiquete y
reservar el cupón. El JSON contiene `id`, `coupon` (texto con ceros iniciales),
`nodeId`, `tir`, `bets` (`nums` como array numérico y `valueCents` entero),
`totalCents`, `status`, `createdBy` y `createdAt` (RFC 3339 en UTC).
`totalCents` se calcula para la respuesta; no es un atributo persistido.

```json
{
  "id": "019a052b-3000-7000-8000-000000000001",
  "coupon": "0007",
  "nodeId": "019a052b-3000-7000-8000-000000000002",
  "tir": 3573,
  "bets": [
    {"nums": [1, 2, 3], "valueCents": 50000},
    {"nums": [10, 20, 30], "valueCents": 25000}
  ],
  "totalCents": 75000,
  "status": "pending",
  "createdBy": "SellerName",
  "createdAt": "2026-10-25T00:00:00Z"
}
```

| HTTP | Situación |
| --- | --- |
| `400` | Query inválida, parámetros desconocidos/repetidos/vacíos, body, UUID o formato de apuestas inválidos |
| `404` | El usuario vendedor no existe |
| `409` | Sorteo cerrado/sorteado, aún no iniciado o fuera del horario de apuestas |
| `503` | Runtime o draw ausente, inválido o no disponible; UUID/TIR discordantes |
| `500` | Error al consultar el usuario, generar el cupón, construir o guardar el tiquete |

El manejo explícito de peticiones `HEAD` queda pendiente para una etapa posterior.

Los errores de entrada, usuario y runtime retornan JSON con `error`; los
internos usan un mensaje genérico. El handler agrega `Cache-Control: no-store`
tanto a errores como a respuestas exitosas. No reintenta automáticamente una
venta fallida ni devuelve éxito ante errores de persistencia.

Los límites técnicos configurables de cantidad de apuestas y longitud de query
quedan pendientes. El servidor usa el buffer predeterminado de Fiber.
Los límites de cliente y proxy deben coordinarse en cada despliegue.

El entorno requiere la tabla configurada por `DYNAMODB_TABLE`, un usuario
replicado, `DRAW_RUNTIME` con `current.id` UUID y `current.tir` positivo y el
draw canónico correspondiente y el contador `COUNTER#TICKET`/`META`
inicializado según el modelo de tiquetes.
La API necesita permisos de lectura, incremento del contador y escritura
transaccional. No crea ni reinicia estos registros automáticamente.
La respuesta confirma la persistencia en estado `pending`; la aceptación
financiera sigue pendiente. La comprobación de estado y horario se realiza
antes del guardado; no es atómica con esa escritura.

### 7.4 Repetición

Repetir un tiquete crea un nuevo tiquete para la partida que se encuentre abierta en ese momento, copia todas sus apuestas y conserva sus valores. La operación se somete nuevamente a validación de horario, configuración y saldo.

### 7.5 Anulación

Un tiquete solo puede anularse antes del cierre de su sorteo. Una anulación válida:

1. excluye definitivamente el tiquete del sorteo y del jackpot;
2. reintegra una sola vez el total apostado al saldo del cajero mediante backoffice;
3. conserva el tiquete y la trazabilidad de la anulación para auditoría.

### 7.6 Estados funcionales

Como mínimo, el sistema debe distinguir conceptualmente:

- aceptado y pendiente de sorteo;
- anulado;
- liquidado sin premio;
- liquidado con premio pendiente de pago;
- pagado.

La implementación puede separar el estado del sorteo, la liquidación y el pago si eso evita transiciones ambiguas.

## 8. Determinación del premio

La cantidad de *spots* es la cantidad de números seleccionados en una apuesta. La cantidad de aciertos es el tamaño de la intersección entre esos números y los 20 números ganadores.

Cada apuesta se liquida de forma independiente. Si la combinación de spots y aciertos aparece en la tabla de pagos, el premio se calcula con su cuota. En cualquier otro caso, el premio es cero.

El premio normal de un tiquete es la suma de los premios de todas sus apuestas. Las apuestas duplicadas se liquidan por separado.

## 9. Representación monetaria y cuotas

Todos los importes se reciben, calculan, almacenan y transmiten como enteros en la unidad mínima de la moneda. No se permite aritmética monetaria con tipos de coma flotante.

Las cuotas se representan como enteros con escala `100`:

- `1.00` se almacena como `100`;
- `3.50` se almacena como `350`;
- `10000.00` se almacena como `1000000`.

La cuota representa el pago total e incluye la apuesta original:

```text
pago_centavos = floor((apuesta_centavos × cuota_escalada) / 100)
```

Toda fracción de la unidad monetaria mínima se redondea hacia abajo.

Ejemplo:

```text
apuesta_centavos = 50000  # 500 unidades monetarias
cuota_escalada   = 350    # x3.5
pago_centavos    = floor((50000 × 350) / 100)
                  = 175000 # 1750 unidades monetarias
```

## 10. Tabla de pagos fija

Las celdas con `—` no pagan premio. Todas las cuotas son multiplicadores del total apostado en esa apuesta.

| Aciertos | 1 spot | 2 spots | 3 spots | 4 spots | 5 spots | 6 spots | 7 spots | 8 spots | 9 spots | 10 spots |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | — | — | — | — | — | — | x1 | x1 | x2 | x2 |
| 1 | x3.5 | x1 | — | — | — | — | — | — | — | — |
| 2 | — | x10 | x2 | x1 | x1 | — | — | — | — | — |
| 3 | — | — | x50 | x10 | x3 | x2 | x2 | — | — | — |
| 4 | — | — | — | x100 | x20 | x15 | x4 | x5 | x1 | — |
| 5 | — | — | — | — | x150 | x60 | x20 | x15 | x10 | x5 |
| 6 | — | — | — | — | — | x500 | x80 | x50 | x25 | x30 |
| 7 | — | — | — | — | — | — | x1000 | x200 | x125 | x100 |
| 8 | — | — | — | — | — | — | — | x2000 | x1000 | x300 |
| 9 | — | — | — | — | — | — | — | — | x5000 | x2000 |
| 10 | — | — | — | — | — | — | — | — | — | x10000 |

## 11. Probabilidades y RTP

Para una apuesta de `s` spots, la probabilidad de obtener exactamente `k` aciertos sigue una distribución hipergeométrica:

```text
P(X = k) = C(s, k) × C(80 - s, 20 - k) / C(80, 20)
```

El RTP teórico de una categoría es:

```text
RTP(s) = Σ P(X = k) × cuota(s, k)
```

La tabla suministrada produce los siguientes valores teóricos antes del efecto de redondear fracciones de la unidad monetaria mínima:

| Spots | RTP teórico |
| ---: | ---: |
| 1 | 87.50% |
| 2 | 98.10% |
| 3 | 97.13% |
| 4 | 95.15% |
| 5 | 86.08% |
| 6 | 93.79% |
| 7 | 93.61% |
| 8 | 92.94% |
| 9 | 82.08% |
| 10 | 90.82% |

La tabla de pagos y su escala son fijas para toda la plataforma. Cualquier cambio futuro exige una nueva versión explícita de las reglas y no puede alterar retroactivamente tiquetes aceptados.

## 12. Jackpot por club

Cada club mantiene un jackpot independiente, aunque todos los clubs compartan el sorteo global.

### 12.1 Configuración

Backoffice proporciona para cada club:

- porcentaje de contribución;
- monto semilla;
- objetivo mínimo;
- objetivo máximo.

Al inicializar el jackpot y después de cada premio, se elige un nuevo objetivo secreto y aleatorio dentro del intervalo inclusivo configurado. El objetivo no se expone a cajeros ni jugadores.

### 12.2 Contribución

Después de liquidar el resultado normal de un club para un sorteo:

```text
ganancia_casa_centavos = max(total_apostado_centavos - premios_normales_centavos, 0)
aporte_jackpot_centavos = floor(ganancia_casa_centavos × porcentaje_configurado)
jackpot_acumulado       = jackpot_anterior + aporte_jackpot_centavos
```

El porcentaje debe representarse mediante un entero escalado y aplicarse con aritmética entera. Si la ganancia de la casa es cero o negativa, el aporte de ese club y sorteo es cero.

### 12.3 Activación y ganador

Cuando el valor acumulado alcanza o supera el objetivo secreto:

1. son elegibles todos los tiquetes válidos, no anulados, del club que participaron en ese sorteo;
2. cada tiquete tiene exactamente la misma probabilidad, sin importar su total apostado, la cantidad de apuestas que contiene o si obtuvo un premio normal;
3. se selecciona un solo tiquete mediante un proceso aleatorio uniforme;
4. el tiquete recibe el valor total acumulado del jackpot, además de cualquier premio normal;
5. el jackpot se reinicia con el monto semilla;
6. se genera un nuevo objetivo secreto entre el mínimo y el máximo configurados.

El jackpot se acredita al saldo del cajero vendedor solamente cuando ese cajero confirma el pago en efectivo al jugador.

La liquidación debe ser idempotente. Un reintento no puede seleccionar otro ganador, modificar el monto ya asignado ni reiniciar dos veces el jackpot.

## 13. Pago de premios

Los tiquetes liquidados con premio aparecen en la aplicación del cajero vendedor con una acción de pago.

Para pagar:

1. el jugador presenta el tiquete;
2. el cajero vendedor lo localiza mediante su número de cupón;
3. Keno4min verifica que el tiquete pertenece a ese cajero, tiene un premio y no fue anulado ni pagado;
4. el cajero entrega en efectivo la suma del premio normal y el jackpot, si aplica;
5. el pago queda registrado de forma irreversible y auditable;
6. backoffice acredita exactamente una vez el valor pagado al saldo del cajero.

Un cajero distinto, incluso dentro del mismo club, no puede pagar el tiquete.

## 14. Requisitos no funcionales

### 14.1 Consistencia e idempotencia

El diseño de idempotencia de creación está aplazado para la etapa actual,
según la sección 7.3.3. Las siguientes reglas siguen siendo requisitos del MVP.

- Cada solicitud de creación, repetición, anulación y pago debe admitir una clave de idempotencia.
- Un sorteo solo puede tener un resultado definitivo.
- Un tiquete solo puede debitarse, reintegrarse y pagarse una vez.
- Un jackpot activado solo puede tener un ganador y un monto definitivos.

### 14.2 Aleatoriedad

- La extracción de números y la selección del ganador del jackpot deben usar un generador criptográficamente seguro y apropiado para certificación en los mercados objetivo.
- No debe existir sesgo por orden, valor, cantidad de apuestas, cajero o identificador del tiquete.
- Deben conservarse evidencias suficientes para reproducir la auditoría del proceso sin permitir predecir resultados futuros.

### 14.3 Auditoría

Debe conservarse un historial inmutable o equivalente de:

- creación, repetición y anulación de tiquetes;
- reglas y configuraciones efectivas en cada operación;
- resultado y tiempos de cada sorteo;
- liquidación de cada apuesta;
- cálculos y movimientos de cada jackpot;
- intentos y confirmaciones de pago;
- correlación con las operaciones financieras de backoffice.

### 14.4 Seguridad

La autenticación y las comprobaciones del vendedor y club se aplazaron en el
flujo de creación actual; la sección 7.3.3 delimita esa etapa. Los siguientes
son requisitos del producto completo.

- Keno4min debe validar la identidad, el club y los permisos efectivos del cajero.
- El cliente nunca puede suministrar resultados, premios calculados, cuotas ni estados confiables.
- Las configuraciones obtenidas de backoffice deben validarse antes de utilizarlas.
- El cupón consecutivo es predecible y permite inferir el avance de las asignaciones; se acepta esa exposición, que no equivale al volumen exacto de ventas por los saltos permitidos. Conocer un UUID o cupón no autoriza consultar, anular ni pagar un tiquete: cada operación debe comprobar la identidad y los permisos efectivos del cajero.

### 14.5 Tiempo

- Los tiempos del servidor son la única autoridad para el cierre de ventas.
- La latencia del cliente no extiende el periodo de venta.
- Los eventos programados retrasados o duplicados no pueden producir sorteos adicionales.

## 15. Criterios de aceptación principales

Estos criterios describen el objetivo del MVP. Las validaciones aplazadas de
creación están indicadas en la sección 7.3.3 y no se consideran implementadas.

1. Dados dos clubs distintos, ambos reciben los mismos 20 números para un sorteo global.
2. Una solicitud recibida en `closesAt` o después no crea ni repite un tiquete para ese sorteo.
3. Un tiquete con varias apuestas se liquida sumando el pago truncado de cada apuesta por separado.
4. Dos apuestas idénticas dentro del mismo tiquete se aceptan y liquidan de forma independiente.
5. Repetir un tiquete genera otro número de cupón y lo asocia con la partida abierta en ese momento.
6. Anular un tiquete antes del cierre lo excluye del sorteo y reintegra su importe una sola vez.
7. Ningún cajero distinto del vendedor puede pagar un tiquete.
8. Pagar dos veces o reintentar el mismo pago no genera un segundo crédito de saldo.
9. Un club con ganancia de casa negativa aporta cero a su jackpot.
10. Cuando un jackpot cruza su objetivo, todos los tiquetes elegibles del club tienen igual probabilidad de ser seleccionados.
11. Un tiquete puede ganar el jackpot aunque no tenga un premio normal.
12. El jackpot de un club no modifica el acumulado ni la configuración de otro club.
13. Reprocesar un evento de sorteo no cambia su resultado, premios ni ganador del jackpot.
14. Cada ejecución respeta el orden `create_draw` → `wait_draw` → `draw` →
    `update_tickets` y no genera las `balls` antes de `closesAt`.
15. Con `closesAt = startsAt + 157 segundos`, una operación recibida en ese
    timestamp o después queda fuera de la ventana de venta del sorteo.
16. `GET /tickets` toma el vendedor del token y recibe las apuestas mediante
    `stv` y `sum`, sin body ni ID del tiquete en la query o como segmento de ruta;
    parámetros adicionales o repetidos, grupos vacíos o
    cantidades diferentes se rechazan.
17. Cada nuevo tiquete recibe un UUID v7 generado por el backend y un
    cupón con padding mínimo de 4 dígitos; solicitudes concurrentes no pueden aceptar
    el mismo cupón.
18. Una reserva de cupón existente impide guardar un segundo tiquete con ese
    cupón. Un fallo al guardar no deja una reserva nueva sin su tiquete, y una
    anulación no libera el cupón para reutilizarlo.
19. Se aceptan huecos en la secuencia. Superar `9999` conserva los dígitos
    adicionales sin reiniciar la secuencia. Alcanzar el máximo acordado del
    contador impide asignar otro cupón sin reiniciar ni reciclar números.
20. El usuario debe existir; el tiquete conserva `CreatedBy` desde
    `user.Username` y `NodeID` desde `user.NodeId` en el momento de la venta.
    No recibe un club independiente desde el frontend ni persiste el UUID
    del usuario.
21. El tiquete conserva sus apuestas, obtiene el total sumando sus valores sin almacenarlo,
    comienza en `pending` y registra la hora UTC del servidor. La estructura
    de creación no incluye `RulesVersion` ni un campo de moneda.
22. El tiquete se asocia al sorteo mediante su `TIR` positivo y único. El
    listado muestra ese número de tiro desde los datos del tiquete, sin
    consultar el sorteo por cada fila ni almacenar su UUID en el tiquete.
23. El tiquete copia `user.Username` a `CreatedBy` al crearse y el listado devuelve ese
    nombre histórico sin consultar al usuario por cada fila. Un cambio de
    username no altera los tiquetes anteriores. No existen campos `UserID` ni
    `Username` en el tiquete y el frontend no suministra `CreatedBy` al crearlo.

## 16. Riesgos y decisiones pendientes

- Coordinar los límites técnicos configurables del endpoint con la longitud máxima admitida por el cliente y los proxies.
- Definir la correlación del vendedor para consultas y pagos al conservar solo su nombre histórico en `CreatedBy`, sin su UUID en el tiquete.
- Definir la fuente de los límites monetarios por club y la comprobación y descuento del saldo antes de completar la venta.
- Incorporar en una etapa posterior las comprobaciones de autenticación, permisos, actividad y pertenencia del club aplazadas en la sección 7.3.3.
- Definir el transporte, alcance y vigencia de la clave de idempotencia de creación, y la recuperación de la respuesta tras un resultado incierto.
- Definir códigos específicos para agotamiento del contador y conflictos de reserva; actualmente devuelven el error interno genérico de creación.
- Validar la capacidad del contador global frente al volumen de asignaciones previsto y definir su procedimiento de recuperación sin reutilizar cupones.
- Definir el vencimiento de los premios y el tratamiento de premios no reclamados.
- Definir qué ocurre con solicitudes que llegan durante el cierre: rechazo definitivo o reasignación explícita al siguiente sorteo.
- Definir el protocolo de compensación cuando backoffice no está disponible.
- Definir la precisión escalada del porcentaje de contribución al jackpot.
- Definir zona horaria, anclaje exacto y tolerancia operativa del schedule.
- Definir la política de concurrencia si `update_tickets` continúa cuando
  EventBridge Scheduler inicia el siguiente intervalo, así como el tratamiento
  de retrasos y acumulación de ejecuciones.
- Definir objetivos de disponibilidad, latencia, retención y recuperación ante desastres.
- Identificar jurisdicciones objetivo y sus requisitos de licencia, certificación, juego responsable y protección de datos.

## 17. Referencias informativas

Estas referencias orientan la completitud y auditabilidad del diseño; no implican certificación ni adopción de una jurisdicción específica.

- [GLI-15: Sistemas de Juegos Bingo y Keno Electrónicos](https://gaminglabs.com/wp-content/uploads/2018/09/GLI-15-V1-3-SP-07122016.pdf)
- [UK Gambling Commission RTS 3: reglas, descripción y probabilidad de ganar](https://www.gamblingcommission.gov.uk/standards/remote-gambling-and-software-technical-standards/rts-3-rules-game-descriptions-and-the-likelihood-of-winning)
- [ISO/IEC/IEEE 29148: ingeniería de requisitos](https://www.iso.org/obp/ui/#iso:std:iso-iec-ieee:29148:ed-2:v1:en)
