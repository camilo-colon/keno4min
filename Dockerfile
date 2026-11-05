# Imagen única con dos targets finales:
#   docker build --target api    -t keno4min-api .
#   docker build --target worker -t keno4min-worker .

FROM golang:1.27-alpine AS build
WORKDIR /src

# Descarga de dependencias en una capa propia para aprovechar la caché.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian13:nonroot AS api
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM gcr.io/distroless/static-debian13:nonroot AS worker
COPY --from=build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
