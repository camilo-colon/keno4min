# Keno4min — Product Requirements Document

| Campo | Valor |
| --- | --- |
| Estado | Borrador |
| Versión | 0.1 |
| Última actualización | 2026-08-29 |
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
- cerrar la venta y repetición de tiquetes tres segundos antes del sorteo;
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
5. El proceso de venta se cierra tres segundos antes del inicio programado.
6. Durante el cierre no se pueden crear, repetir ni anular tiquetes para ese sorteo.
7. Un disparador programado de Amazon EventBridge inicia el procesamiento cada tres minutos.
8. La ejecución debe ser idempotente: el mismo sorteo no puede generar dos resultados ni liquidarse dos veces, aunque el evento sea entregado o reintentado más de una vez.

## 7. Tiquetes y apuestas

### 7.1 Estructura conceptual

Cada tiquete:

- pertenece a un único sorteo, club y cajero;
- contiene una o más apuestas;
- registra la moneda heredada del administrador;
- recibe un número de cupón aleatorio de 16 caracteres, único en toda la plataforma;
- conserva una copia inmutable de las apuestas aceptadas y de las reglas económicas necesarias para su liquidación.

Ejemplo conceptual de apuestas:

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

Un tiquete puede contener apuestas idénticas si el jugador así lo solicita. No existe un límite funcional de apuestas por tiquete en el MVP; la API deberá aplicar un límite técnico configurable para proteger el servicio.

### 7.3 Creación

El total del tiquete es la suma de `value_cents` de todas sus apuestas. El tiquete solo queda aceptado si Keno4min puede garantizar el débito único de ese total en el saldo del cajero.

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

- Keno4min debe validar la identidad, el club y los permisos efectivos del cajero.
- El cliente nunca puede suministrar resultados, premios calculados, cuotas ni estados confiables.
- Las configuraciones obtenidas de backoffice deben validarse antes de utilizarlas.
- Los identificadores públicos no deben permitir enumerar tiquetes ni inferir el volumen de ventas.

### 14.5 Tiempo

- Los tiempos del servidor son la única autoridad para el cierre de ventas.
- La latencia del cliente no extiende el periodo de venta.
- Los eventos programados retrasados o duplicados no pueden producir sorteos adicionales.

## 15. Criterios de aceptación principales

1. Dados dos clubs distintos, ambos reciben los mismos 20 números para un sorteo global.
2. Una solicitud recibida dentro de los tres segundos previos al sorteo no crea ni repite un tiquete para ese sorteo.
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

## 16. Riesgos y decisiones pendientes

- Definir el límite técnico de apuestas por tiquete y el tamaño máximo de una solicitud.
- Definir el alfabeto del número de cupón y la estrategia ante colisiones.
- Definir el vencimiento de los premios y el tratamiento de premios no reclamados.
- Definir qué ocurre con solicitudes que llegan durante el cierre: rechazo definitivo o reasignación explícita al siguiente sorteo.
- Definir el protocolo de compensación cuando backoffice no está disponible.
- Definir la precisión escalada del porcentaje de contribución al jackpot.
- Definir zona horaria, anclaje exacto y tolerancia operativa del schedule.
- Definir objetivos de disponibilidad, latencia, retención y recuperación ante desastres.
- Identificar jurisdicciones objetivo y sus requisitos de licencia, certificación, juego responsable y protección de datos.

## 17. Referencias informativas

Estas referencias orientan la completitud y auditabilidad del diseño; no implican certificación ni adopción de una jurisdicción específica.

- [GLI-15: Sistemas de Juegos Bingo y Keno Electrónicos](https://gaminglabs.com/wp-content/uploads/2018/09/GLI-15-V1-3-SP-07122016.pdf)
- [UK Gambling Commission RTS 3: reglas, descripción y probabilidad de ganar](https://www.gamblingcommission.gov.uk/standards/remote-gambling-and-software-technical-standards/rts-3-rules-game-descriptions-and-the-likelihood-of-winning)
- [ISO/IEC/IEEE 29148: ingeniería de requisitos](https://www.iso.org/obp/ui/#iso:std:iso-iec-ieee:29148:ed-2:v1:en)
