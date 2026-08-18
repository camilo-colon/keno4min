# Producto

## Hechos confirmados

- Keno4min es un servicio interno.
- Sus capacidades iniciales son tickets y sorteos de Keno4min.
- Un gateway sera el unico punto autorizado para acceder al servicio y asumira
  la autenticacion del consumidor.
- A 2026-08-18 no hay endpoints ni reglas de negocio implementadas.

Estos hechos provienen del README y de ADR 0001. Este documento no completa los
vacios con supuestos.

## Decision pendiente: confianza entre gateway y servicio

Que el gateway autentique al consumidor no define como el servicio comprobara
que una solicitud proviene realmente del gateway ni como recibira una identidad
confiable. Antes de implementar identidad se deben decidir y documentar:

- la autenticacion entre gateway y servicio;
- los encabezados o claims aceptados y quien puede establecerlos;
- la proteccion contra suplantacion y replay;
- la propagacion de actor, tenant e identificador de solicitud;
- el comportamiento ante identidad ausente o invalida.

No se debe confiar en un encabezado solo por su nombre o por provenir de una red
interna.

## Decisiones necesarias antes del primer flujo vertical

El primer product spec debe definir, como minimo:

- actores y permisos;
- ciclo de vida y estados de un ticket;
- seleccion, limites y validacion de jugadas;
- calendario, zona horaria y cierre de cada sorteo;
- generacion, publicacion y correccion de resultados;
- calculo o consulta de premios;
- garantias de idempotencia y politica de reintentos;
- retencion, auditoria y datos sensibles;
- comportamiento ante retrasos, duplicados y concurrencia.

## Formato de una especificacion

Las nuevas especificaciones de producto deben vivir en `docs/product-specs/`
cuando exista la primera. Cada una incluye:

1. Problema y resultado esperado.
2. Actores y autorizacion.
3. Reglas e invariantes con ejemplos.
4. Estados y transiciones.
5. Casos limite y fallos esperados.
6. Criterios de aceptacion observables.
7. Decisiones fuera de alcance.

No se crea el directorio hasta tener una especificacion concreta, de acuerdo con
ADR 0001.
