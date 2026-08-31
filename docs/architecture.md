# Arquitectura

Keno4min usa *vertical slices*: cada funcionalidad mantiene juntos su dominio,
casos de uso y adaptadores. Consulta el [ADR 0004](adrs/0004-organizar-el-codigo-por-vertical-slices.md).

## Estructura

```text
cmd/{api,worker}/             # Arranque e inyección de dependencias
internal/
├── config/                   # Variables de entorno
├── platform/awsclient/       # Clientes técnicos compartidos
└── <slice>/                  # Una funcionalidad, por ejemplo draw
    ├── <slice>.go, port.go   # Dominio y contratos
    ├── handler/, service/    # Entradas y casos de uso
    └── infrastructure/       # Persistencia e integraciones
```

Las carpetas se crean solo cuando se necesitan; pruebas y mocks viven junto al
código que verifican.

## Dependencias

```text
cmd ──> handler ──> service ──> dominio y puertos
 ├──> infrastructure ─────────> dominio y puertos
 └──> platform ───────────────> SDKs externos
```

- El dominio y los puertos no importan Fiber, AWS ni infraestructura.
- `service` no depende de `handler` ni de `infrastructure`.
- `cmd` construye e inyecta las implementaciones concretas.
- `platform` comparte clientes técnicos, pero no contiene lógica de negocio.
