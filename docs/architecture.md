# Arquitectura

Keno4min se organiza por funcionalidad. Cada módulo contiene su dominio y casos
de uso, con adaptadores separados para HTTP y DynamoDB. Consulta el
[ADR 0008](adrs/0008-organizar-por-funcionalidad-y-adaptadores.md).

## Estructura

```text
cmd/api/                     # Arranque HTTP e inyección de dependencias
cmd/worker/                  # Punto de entrada base, sin flujo implementado
internal/
├── config/                  # Variables de entorno
├── platform/aws/            # Configuración base del SDK AWS y módulo Fx
├── platform/dynamodb/       # Cliente DynamoDB y módulo Fx
├── platform/server/         # Servidor HTTP y middleware transversal
└── draw/
    ├── draw.go              # Entidad y reglas de negocio
    ├── get_by_id.go         # Caso de uso y contrato IDReader
    ├── dynamodb/
    │   ├── store.go         # Lectura del draw canónico mediante GetItem
    │   └── item.go          # Formato almacenado y decodificación
    └── http/
        └── handler.go      # Ruta por ID, respuesta JSON y errores HTTP
```

Las pruebas viven junto al código que verifican. Los paquetes y archivos se
crean cuando se implementa una responsabilidad, sin anticipar operaciones.

## Dependencias

- `draw` contiene la entidad, sus invariantes, los casos de uso y los contratos
  pequeños que estos consumen. No importa Fiber, AWS ni sus adaptadores.
- `draw/http` depende de `draw` y Fiber. Adapta parámetros, configura el plazo de
  la petición y construye respuestas HTTP, incluidos nombres JSON, UTC y bolas
  como array numérico.
- `draw/dynamodb` depende de `draw` y el SDK AWS. Construye claves, lee y convierte
  los datos, comprobando el formato de almacenamiento. Puede compartir esta
  implementación entre operaciones del módulo.
- Los adaptadores no se importan entre sí. Un adaptador puede implementar varios
  contratos pequeños; no se introduce un repositorio CRUD genérico.
- `cmd/api` construye el cliente AWS, el store, el caso de uso y el handler, y
  registra la ruta. `platform` comparte infraestructura técnica, no negocio.

### Configuración compartida de AWS

`platform/aws` proporciona un único `aws.Config` por aplicación Fx mediante
`config.LoadDefaultConfig`, con un plazo de carga de diez segundos. El SDK
resuelve región y credenciales desde sus fuentes habituales, incluidas las
variables `AWS_REGION`, `AWS_PROFILE` y los roles del entorno de ejecución.

`platform/dynamodb` recibe esa configuración base y `config.Config` para
construir el cliente. Aplica `DYNAMODB_ENDPOINT` únicamente al cliente DynamoDB;
los futuros clientes AWS podrán consumir la misma configuración base.

`cmd/api` registra ambos módulos junto a los de funcionalidad y servidor. El
repositorio de `draw` recibe el cliente y `config.Config`, y conserva el nombre
de la tabla definido por `DYNAMODB_TABLE`. La configuración compartida de AWS
no contiene nombres de tablas ni reglas de persistencia del negocio.

## Consulta implementada

`GET /api/v1/draws/{id}` ejecuta `draw.GetByID`. El caso de uso valida el parámetro,
lee mediante `IDReader`, invoca `Draw.Validate()` y comprueba que el draw tenga
el ID solicitado. Devuelve la entidad validada al adaptador HTTP.

`Draw.Validate()` centraliza las reglas de identidad, tiempos, estado y bolas.
`draw.New` usa la misma validación al crear un sorteo. Los errores de datos
almacenados se distinguen de los errores de parámetros de la petición.

La consulta por TIR anterior fue retirada. Las consultas por TIR, histórico y
runtime conservan su diseño de acceso documentado, pero no tienen implementación.
Las futuras consultas del draw completo pueden reutilizar el dominio y la
persistencia; las proyecciones parciales mantienen sus contratos específicos.

Los endpoints de sorteos son de lectura. Las tareas coordinadas por Step
Functions producirán el estado canónico y sus proyecciones; el arranque de
worker sigue siendo una base sin flujo implementado.
