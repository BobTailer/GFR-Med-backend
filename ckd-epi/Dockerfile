# Билд стадии
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем swag для генерации Swagger документации
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Копируем исходный код
COPY . .

# Генерируем Swagger документацию (используем полный путь к swag)
ENV PATH="${PATH}:/go/bin"
RUN swag init -g internal/api/server.go -o ./docs

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
# Копируем сгенерированную Swagger документацию
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./ckd-epi"]

