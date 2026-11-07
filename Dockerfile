# Imagen única con dos targets finales:
#   docker build --target api    -t keno4min-api .
#   docker build --target worker -t keno4min-worker .
# La imagen siempre es linux/amd64 (runtime_platform X86_64 en ECS): el stage
# de build compila de forma cruzada en la plataforma nativa del host y los
# stages finales fijan la plataforma de la imagen base.

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src

# Descarga de dependencias en una capa propia para aprovechar la caché.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM --platform=linux/amd64 gcr.io/distroless/static-debian13:nonroot AS api
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM --platform=linux/amd64 gcr.io/distroless/static-debian13:nonroot AS worker
COPY --from=build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
