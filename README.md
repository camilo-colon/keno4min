# Keno4min

Backend en Go para un juego de Keno independiente orientado inicialmente a puntos de venta. El producto contempla sorteos globales cada tres minutos, tiquetes con múltiples apuestas, liquidación mediante una tabla de pagos fija y jackpots independientes por club.

> [!IMPORTANT]
> El repositorio se encuentra en una etapa inicial. Actualmente contiene la base de la API HTTP; el motor de sorteos, los tiquetes, la liquidación y los jackpots descritos en el PRD todavía no están implementados.

## Documentación

- [Product Requirements Document](docs/product-requirements.md): alcance del MVP, reglas, tabla de pagos, RTP, jackpot y criterios de aceptación.
- [Arquitectura](docs/architecture.md): organización del código y reglas de dependencias.
- [OpenAPI v1](api/openapi/v1/openapi.yaml): contrato actual de la API HTTP.
- [Architecture Decision Records](docs/adrs/README.md): decisiones técnicas del proyecto.

## Estado técnico

El servicio utiliza Go y Fiber v3. La base actual incluye configuración mediante variables de entorno, endpoints de salud, identificadores de solicitud, respuestas de error basadas en Problem Details, CORS, cabeceras de seguridad y apagado controlado.

## Ejecutar la API

```sh
go run ./cmd/api
```

La configuración usa variables de entorno; consulta [`.env.example`](.env.example) para ver los valores disponibles. El servidor escucha en `:8080` por defecto.

## AWS y DynamoDB

La tabla, sus índices y los permisos IAM se administran en el repositorio
central de infraestructura. Keno4min recibe únicamente la configuración de
runtime:

- `AWS_REGION`: región donde está desplegada la tabla.
- `DYNAMODB_TABLE_NAME`: nombre físico de la tabla asignada al entorno.
- `DYNAMODB_ENDPOINT_URL`: endpoint alternativo opcional, permitido solamente
  en desarrollo y pruebas.

En `development` y `test`, la región y tabla usan `us-east-1` y
`keno4min-local` como valores predeterminados. En los demás entornos ambos
valores son obligatorios y los endpoints personalizados se rechazan.

Las credenciales no forman parte de la configuración propia de la aplicación.
Cuando se incorpore el cliente de AWS, este utilizará la cadena estándar de
credenciales del SDK, incluidos perfiles locales y roles IAM del workload.
Esta configuración inicial no realiza conexiones a AWS ni comprueba la
existencia de la tabla.

El cliente se construye mediante `internal/platform/awsclient`. La fábrica
carga la cadena estándar de configuración del AWS SDK, aplica la región y, en
entornos locales, configura el endpoint específico de DynamoDB. Crear el cliente
no realiza solicitudes de red; el primer adaptador que lo necesite lo conectará
desde `cmd` de esta forma:

```go
client, err := awsclient.NewDynamoDB(ctx, cfg.AWS, cfg.DynamoDB)
if err != nil {
    // El proceso debe terminar porque la configuración del SDK es inválida.
}

repository := drawdynamodb.New(client, cfg.DynamoDB.TableName)
```

El repositorio del ejemplo todavía no está implementado. El cliente se creará
una sola vez por proceso y se compartirá mediante inyección de dependencias.

## Endpoints iniciales

- `GET /`: identidad del servicio.
- `GET /health/live`: liveness probe.
- `GET /health/ready`: readiness probe.
- `GET /api/v1`: raíz de la API v1.

## Verificación

```sh
go test ./...
```
