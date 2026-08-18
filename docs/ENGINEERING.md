# Convenciones de ingenieria

## Go

- La version minima es la declarada en `go.mod`.
- Usa `gofmt`; nombres exportados y errores siguen las convenciones de Go.
- Envuelve errores con contexto usando `%w`; compara causas con `errors.Is` o
  `errors.As`.
- Propaga `context.Context` por operaciones que hacen I/O. No lo guardes en
  structs ni lo sustituyas por `context.Background()` dentro de un request.
- Prefiere dependencias pequenas y legibles. Una dependencia nueva requiere una
  necesidad concreta; no existe una prohibicion de librerias externas.
- Evita estado global mutable. Inyecta reloj, generadores de ID o adaptadores
  cuando la prueba o la consistencia lo requieran.

## Paquetes

- Un paquete representa una responsabilidad, no una categoria de archivos.
- Las interfaces se definen del lado consumidor y permanecen pequenas.
- `cmd/api` ensambla; los casos de uso viven en `internal/ticket` o
  `internal/draw`.
- Un adaptador puede conocer al negocio; el negocio no conoce al adaptador.
- Elimina `.gitkeep` al agregar el primer archivo real a una carpeta.

## HTTP y OpenAPI

Sigue ADR 0002. El orden de trabajo es:

1. Definir operacion, esquemas, errores y ejemplos en OpenAPI.
2. Validar que el cambio sea compatible con la version mayor.
3. Generar o adaptar los tipos de transporte.
4. Traducir explicitamente entre transporte y negocio.
5. Probar contrato, traduccion y comportamiento.

Problem Details se construye en HTTP. Un error de dominio comunica significado
del negocio, no un status code.

## Persistencia

- Repositorios expresan operaciones del caso de uso, no una API CRUD generica.
- Cada indice tiene una consulta o invariante que justifica su existencia.
- Escrituras que preservan una invariante deben ser atomicas en el nivel correcto.
- Las migraciones son versionadas, observables y reanudables o tienen un plan de
  recuperacion explicito.
- BSON, documentos MongoDB y DTO HTTP no atraviesan el dominio.

## Pruebas

- Ubica tests junto al paquete y nombra el comportamiento que demuestran.
- Usa tablas cuando aclaren variantes; evita tablas que oculten escenarios.
- Prueba adaptadores con la implementacion real o un sustituto que preserve su
  semantica cuando el riesgo esta en la integracion.
- No pruebes solo mocks: conserva al menos una prueba del borde real para cada
  integracion critica cuando exista infraestructura para ejecutarla.
- Un bug corregido empieza con una reproduccion automatizada cuando sea viable.

## Observabilidad

- Los logs son estructurados y describen eventos, no volcados de objetos.
- Propaga un identificador de solicitud definido por el borde confiable.
- Metricas y trazas usan nombres estables y cardinalidad acotada.
- Nunca registres credenciales, encabezados completos ni contenido sensible de
  tickets.
