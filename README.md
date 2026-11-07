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

El servidor escucha en `:8080` por defecto; consulta [Configuración](#configuración).

## Configuración

La configuración se lee de variables de entorno (o de un `.env` opcional en
desarrollo). La tabla, las colas, sus índices y los permisos IAM se
administran en el repositorio central de infraestructura, que inyecta estos
valores en ECS. El esquema requerido de `GSI1` y `GSI2` se documenta en el
[modelo de tiquetes](docs/data-model/ticket-access-patterns.md#índices-de-consulta).

| Variable | Uso | Valor por defecto |
| --- | --- | --- |
| `HTTP_ADDRESS` | Dirección de escucha de la API. | `:8080` |
| `ENVIRONMENT` | Entorno de despliegue. Debe ser igual al entorno de infraestructura (`dev`), porque el worker construye con él los nombres de las colas, por ejemplo `cronos-dev-keno4min-node-create_on_node_created`. | `development` |
| `DYNAMODB_TABLE` | Nombre físico de la tabla asignada al entorno. | — |
| `AWS_REGION` | Región de la tabla y las colas; la lee la cadena estándar del AWS SDK. | — |
| `CORE_BASE_URL` | URL base del servicio core, dueño del saldo del nodo. Obligatoria para la API. | — |
| `CORE_TIMEOUT` | Timeout de las llamadas a core, en formato Go (`5s`, `500ms`). | `5s` |
| `DYNAMODB_ENDPOINT` | Endpoint alternativo opcional para desarrollo y pruebas. | — |
| `APP_NAME` | Nombre de la aplicación Fiber. | `Cronos Keno4min API` |

Consulta [`.env.example`](.env.example) para un ejemplo local. Las credenciales
no forman parte de la configuración propia: el AWS SDK usa su cadena estándar,
incluidos perfiles locales y el rol IAM de la tarea en ECS.

## Endpoints

Todas las rutas se publican bajo el prefijo `/keno4min`, porque el ALB interno
compartido y API Gateway reenvían la ruta completa sin reescribirla.

- `GET /keno4min/health`: liveness check del target group del ALB. Responde
  `200` con `{"status":"ok"}`, no requiere `Authorization` y no consulta
  DynamoDB ni core.

Las demás rutas requieren `Authorization: Bearer <token>` y responden `401`
sin él:

- `GET /keno4min/draws/runtime`: estado actual del ciclo de sorteos.
- `GET /keno4min/tickets/create`, `POST /keno4min/tickets/lookup`,
  `GET /keno4min/tickets`, `GET /keno4min/tickets/{id}`,
  `POST /keno4min/tickets/{id}/repeat`, `POST /keno4min/tickets/{id}/cancel`,
  `POST /keno4min/tickets/{id}/pay`: venta, consulta, repetición, anulación y
  pago de tiquetes.

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
