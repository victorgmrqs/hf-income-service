# ================================
# Stage 1: Build
# ================================
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copia go.mod primeiro (melhor uso de cache de dependências)
COPY go.mod ./
# COPY go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Build do binário estático (CGO desabilitado para Alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# ================================
# Stage 2: Runtime
# ================================
FROM alpine:3.20

WORKDIR /app

# Usuário non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copia apenas o binário do stage anterior
COPY --from=builder /app/server .

RUN chown appuser:appgroup /app/server

USER appuser

EXPOSE 8081

CMD ["./server"]
