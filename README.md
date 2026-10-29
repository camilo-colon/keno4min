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
central de infraestructura. El esquema requerido de `GSI1` y `GSI2` se
documenta en el
[modelo de tiquetes](docs/data-model/ticket-access-patterns.md#índices-de-consulta).
Keno4min recibe únicamente la configuración de runtime:

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

## Generar infraestructura de eventos

```sh
go run ./cmd/script
```

Ejecuta el comando desde la raíz del repositorio. Lee los suscriptores
registrados en Fx y regenera `infra/eventbridge/generated/`:

- `sqs_queue_<nombre>.tf`: cola, DLQ y permiso para recibir eventos.
- `event_rule_<nombre>.tf`: regla por patrón de evento.
- `event_target_<regla>.tf`: destinos SQS de cada regla.
- `variables.tf`: nombre del bus existente (`event_bus_name`) y ambiente (`environment`).
- `versions.tf`: requisito del provider AWS del módulo generado.

Las plantillas están en `cmd/script/templates`. El script conserva el arranque
actual con `.env` y configuración AWS, pero solo escribe archivos; no crea
recursos. Los eventos exactos y los prefijos con `*` final se traducen a filtros
sobre `detail-type`. La DLQ conserva `maxReceiveCount = 3`.

`infra/eventbridge` es el proyecto Terraform que consume el módulo generado.
Requiere Terraform >= 1.6 y credenciales AWS disponibles, por ejemplo mediante
`AWS_PROFILE`. La primera vez, prepara los valores del despliegue:

```sh
cp infra/eventbridge/terraform.tfvars.example infra/eventbridge/terraform.tfvars
```

Edita `terraform.tfvars` con tu región, ambiente y nombre del bus existente.
Terraform no carga el `.env` de la aplicación. Luego, desde la raíz:

```sh
go run ./cmd/script &&
cd infra/eventbridge &&
terraform init &&
terraform plan

# Después de revisar el plan:
terraform apply
```

El proyecto configura el provider AWS 6.x y usa estado local en
`infra/eventbridge/terraform.tfstate`. Conserva ese estado entre ejecuciones;
para compartir despliegues configura un backend remoto en el proyecto raíz.
No cambies de ambiente reutilizando el mismo estado: usa estados separados.
Versiona `.terraform.lock.hcl`, creado por `terraform init`, para fijar la
versión instalada del provider. El estado, `.terraform/`, los valores locales
y la salida generada están ignorados por Git.

Bus y colas deben estar en la misma cuenta y región. El bus no se crea aquí.
El ambiente se inserta después de `cronos-` en los nombres físicos de colas, DLQ
y reglas; por ejemplo, `cronos-dev-keno4min-node-created`. Los nombres de archivo y las
referencias Terraform permanecen iguales. Estos valores se asignan al desplegar,
no al ejecutar el generador.
Cambiar el ambiente de recursos ya desplegados cambia sus nombres y puede
reemplazar las colas; revisa el plan antes de aplicar.
Si las colas ya existen y no están administradas por otro estado, impórtalas
usando las direcciones `module.subscriptions.aws_sqs_queue.<nombre>` y
`module.subscriptions.aws_sqs_queue.<nombre>-dl` y sus URL como ID.
Si ya las administra otro proyecto Terraform, migra su estado antes de aplicar;
no administres los mismos recursos desde dos estados.

Cada ejecución elimina `generated/` completa y la vuelve a crear. No guardes
archivos manuales ni estado dentro de esa carpeta. Eliminar un suscriptor
retira sus definiciones; el siguiente plan propondrá eliminar sus recursos,
conservando las reglas que todavía usan otros suscriptores. Solo `apply`
modifica AWS. Si el script falla, no ejecutes Terraform hasta regenerar con
éxito, porque la salida podría quedar incompleta. Modifica las plantillas en
`cmd/script/templates` para cambiar las definiciones generadas.
