# Arquitectura

Keno4min se organiza por funcionalidades mediante *vertical slices*. La razón
de esta elección está registrada en el
[ADR 0004](adrs/0004-organizar-el-codigo-por-vertical-slices.md).

## Organización

Cada slice puede incorporar esta estructura según la necesite:

```text
internal/<slice>/
├── <slice>.go       # Modelo y reglas del dominio
├── port.go          # Contratos de entrada y salida
├── handler/         # Adaptadores de entrada, como HTTP o eventos
├── service/         # Casos de uso, uno por archivo
├── infrastructure/  # Persistencia e integraciones externas
└── mock/            # Dobles reutilizables para pruebas
```

Los slices se incorporan de forma incremental. Actualmente el proyecto comienza
con `draw`; los demás se crearán cuando se implemente su primer caso de uso.

## Dependencias

```text
handler ────────────> dominio y puertos
service ────────────> dominio y puertos
infrastructure ─────> dominio y puertos
```

- `service` no depende de `handler` ni de `infrastructure`.
- Fiber, AWS y otros detalles externos no forman parte del dominio.
- `cmd` crea las implementaciones y conecta sus dependencias.
- Solo se crean las carpetas que cada slice utilice.
- Las pruebas permanecen junto al código que verifican; `mock` es opcional.
