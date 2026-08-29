# Keno4min

Servicio HTTP en Go sobre Fiber v3.

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
