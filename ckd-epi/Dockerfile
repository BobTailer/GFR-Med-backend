# Билд стадии
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ckd-epi ./cmd/ckd-epi

# Финальная стадия
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /app/ckd-epi .
# Копируем шаблоны и ресурсы
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/resources ./resources

EXPOSE 8080

CMD ["./ckd-epi"]

