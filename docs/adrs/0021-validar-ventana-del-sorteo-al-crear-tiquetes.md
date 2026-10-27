# 0021. Validar la ventana del sorteo al crear tiquetes

- Estado: Aceptada
- Fecha: 2026-10-24
- Responsables: Equipo de cronos
- Reemplaza: [ADR 0020](0020-conectar-creacion-http-de-tiquetes.md)

## Contexto

La creación resuelve el sorteo desde runtime, pero esa proyección no garantiza
que el sorteo esté abierto ni dentro del horario de apuestas.

## Decisión

Consultar el draw canónico por su UUID con `draws.Repository.FindByID`, una
lectura fuerte. Antes de generar el cupón, llamar a `Draw.AcceptsTicketsAt`
con la hora UTC del servidor. El método exige `open` y el intervalo
`startsAt <= ahora < closesAt`. Comprobar que UUID y TIR coincidan con runtime.

La validación se realiza en el caso de uso. Por la simplificación solicitada,
el guardado conserva únicamente los dos `Put` de tiquete y cupón, sin otra
consulta ni condición transaccional sobre el sorteo.

Se conservan las demás decisiones del ADR 0020 sobre composición Fx, rutas,
respuesta HTTP y aplazamientos financieros, de permisos e idempotencia.

## Consecuencias

- Un sorteo cerrado, sorteado, futuro o vencido devuelve `409` sin generar un
  cupón ni guardar. Un draw ausente, inválido o discordante devuelve `503`.
- Las reglas de horario se reutilizan desde el dominio Draw.
- La validación previa y el guardado no son atómicos; un cambio posterior a
  la consulta queda fuera de esta comprobación.

## Referencias

- [PRD: creación](../product-requirements.md#73-creación).
- [Modelo de acceso de draws](../data-model/draw-access-patterns.md).
