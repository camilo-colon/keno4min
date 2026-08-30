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

## Endpoints iniciales

- `GET /`: identidad del servicio.
- `GET /health/live`: liveness probe.
- `GET /health/ready`: readiness probe.
- `GET /api/v1`: raíz de la API v1.

## Verificación

```sh
go test ./...
```
