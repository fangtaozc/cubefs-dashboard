FROM node:14-bullseye-slim AS frontend-builder

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.20-bullseye AS backend-builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY backend/ ./backend/
COPY depends/ ./depends/
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

WORKDIR /src/backend
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=vendor -o /out/cfs-gui main.go

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=backend-builder /out/cfs-gui /app/cfs-gui
COPY --from=backend-builder /src/frontend/dist /app/dist
COPY backend/conf/config.yml /app/config/config.yml

EXPOSE 6007

ENTRYPOINT ["/app/cfs-gui"]
CMD ["-c", "/app/config/config.yml"]
