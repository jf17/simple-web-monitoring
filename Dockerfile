# Используем официальный образ Go для сборки
FROM golang:1.21-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY main.go ./

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web-monitor .

# Используем минимальный образ для запуска
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates

# Создаем пользователя для безопасности
RUN adduser -D -s /bin/sh appuser

# Создаем рабочую директорию
WORKDIR /app

# Копируем исполняемый файл из builder
COPY --from=builder /app/web-monitor .

# Создаем директорию для данных и назначаем права
RUN mkdir -p /app/data && chown appuser:appuser /app/data

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт (будет переопределен через docker-compose)
EXPOSE 8080

# Запускаем приложение
ENTRYPOINT ["./web-monitor"]