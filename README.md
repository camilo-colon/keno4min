# Keno4min

Backend en Go para un juego de Keno independiente orientado inicialmente a puntos de venta. El producto contempla sorteos globales cada tres minutos, tiquetes con múltiples apuestas, liquidación mediante una tabla de pagos fija y jackpots independientes por club.

> [!IMPORTANT]
> El repositorio se encuentra en una etapa inicial. Incluye la base HTTP y la consulta de sorteos por ID en DynamoDB. El histórico y runtime aún no están implementados; las escrituras del ciclo se ejecutarán desde las tareas de Step Functions.

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
El cliente AWS utiliza la cadena estándar de
credenciales del SDK, incluidos perfiles locales y roles IAM del workload.
Esta configuración inicial no realiza conexiones a AWS ni comprueba la
existencia de la tabla.

El cliente se construye mediante `internal/platform/awsclient`. La fábrica
carga la cadena estándar de configuración del AWS SDK, aplica la región y, en
entornos locales, configura el endpoint específico de DynamoDB. Crear el cliente
no realiza solicitudes a DynamoDB. `cmd/api` lo conecta al caso de uso y sus adaptadores:

```go
// draw: internal/draw
// drawdynamodb: internal/draw/dynamodb
// drawhttp: internal/draw/http
client, err := awsclient.NewDynamoDB(ctx, cfg.AWS, cfg.DynamoDB)
if err != nil {
    // El proceso debe terminar porque la configuración del SDK es inválida.
}

store := drawdynamodb.NewStore(client, cfg.DynamoDB.TableName)
query := draw.NewGetByID(store)
httpServer.App().Get("/api/v1/draws/:id", drawhttp.NewHandler(query))
```

El cliente se crea una sola vez por proceso y se comparte mediante inyección de
dependencias. La consulta requiere `dynamodb:GetItem` sobre la tabla configurada.

## Endpoints iniciales

- `GET /`: identidad del servicio.
- `GET /health/live`: liveness probe.
- `GET /health/ready`: readiness probe.
- `GET /api/v1`: raíz de la API v1.
- `GET /api/v1/draws/{id}`: sorteo por ID, mediante una lectura fuerte.
  Retorna `400` para un ID en blanco o con codificación URL inválida, `404` si no existe y `500` ante errores
  internos. Consulta el [contrato de almacenamiento](docs/data-model/draw-access-patterns.md#contrato-implementado-para-la-consulta-por-id)
  para los ítems que deben producir las tareas del workflow.

## Verificación

```sh
go test ./...
```
