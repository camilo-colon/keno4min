# Verificacion

## Comando canonico

```bash
./init.sh
```

Es el mismo punto de entrada usado por CI. Actualmente verifica:

- presencia y tamano del mapa de conocimiento;
- indice de ADR y estructura de ExecPlans;
- limites basicos de paquetes y nombres genericos prohibidos;
- forma minima del contrato OpenAPI;
- consistencia de `go.mod`;
- `gofmt`, `go vet` y `go test ./...` cuando existen paquetes Go.

El check de OpenAPI es estructural, no un validador semantico completo. Antes de
publicar el primer endpoint debe agregarse y fijarse una herramienta de lint de
OpenAPI. Hasta entonces, la verificacion no afirma correccion semantica completa
del contrato; el trabajo debe registrarse en un ExecPlan o issue concreto.

## Ciclo durante desarrollo

Ejecuta la comprobacion mas pequena que pueda falsar tu hipotesis y amplia al
final. Ejemplos futuros:

```bash
go test ./internal/ticket
go test -race ./internal/...
./init.sh
```

`-race` debe usarse en cambios con goroutines, caches, estado compartido o
coordinacion concurrente. Puede incorporarse a CI cuando existan esos caminos y
su costo sea conocido.

## Evidencia requerida

El resumen del cambio o su ExecPlan registra:

- comando exacto;
- resultado y fecha;
- pruebas omitidas y razon;
- limitaciones o riesgo residual.

Una frase como "deberia funcionar" no es evidencia. Para cambios HTTP, conserva
un ejemplo reproducible de request/response. Para migraciones, registra prueba
con datos representativos y recuperacion. Para bugs, demuestra primero el fallo
y luego la correccion.

## Definicion de terminado

Un cambio se considera terminado solo si pasa `CHECKPOINTS.md`, actualiza sus
fuentes de verdad y deja el repositorio ejecutable desde un checkout limpio. Un
check preexistente que falle debe quedar identificado; no se oculta ni se
silencia para obtener verde.
