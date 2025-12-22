.PHONY: run build clean test help docker-build docker-compose docker-up docker-down

# Показать справку
help:
	@echo "Доступные команды:"
	@echo "  make run          - Запуск на порту 8080"
	@echo "  make run-port PORT=3000 - Запуск на указанном порту"
	@echo "  make build        - Сборка приложения"
	@echo "  make test         - Тестирование API на порту 8080"
	@echo "  make test-port PORT=3000 - Тестирование API на указанном порту"
	@echo "  make clean        - Очистка"
	@echo ""
	@echo "Docker команды:"
	@echo "  make docker-build - Сборка Docker образа"
	@echo "  make docker-compose PORT=8080 - Создание docker-compose.yml"
	@echo "  make docker-up PORT=8080 - Запуск через Docker Compose"
	@echo "  make docker-down  - Остановка Docker Compose"
	@echo ""
	@echo "Прямой запуск:"
	@echo "  go run main.go -port=8080"

# Запуск приложения (по умолчанию на порту 8080)
run:
	go run main.go -port=8080

# Запуск на указанном порту
run-port:
	@echo "Использование: make run-port PORT=3000"
	@if [ -z "$(PORT)" ]; then echo "Ошибка: укажите PORT=номер_порта"; exit 1; fi
	go run main.go -port=$(PORT)

# Сборка приложения
build:
	go build -o web-monitor main.go

# Очистка
clean:
	rm -f web-monitor
	rm -f docker-compose.yml

# Тестирование API
test:
	@echo "Тестирование API..."
	@curl -s http://localhost:8080/api/services || echo "Сервер не запущен на порту 8080"

# Тестирование API на указанном порту
test-port:
	@echo "Тестирование API на порту $(PORT)..."
	@if [ -z "$(PORT)" ]; then echo "Ошибка: укажите PORT=номер_порта"; exit 1; fi
	@curl -s http://localhost:$(PORT)/api/services || echo "Сервер не запущен на порту $(PORT)"

# Добавление тестового сервиса
add-test-service:
	curl -X POST -H "Content-Type: application/json" \
		-d '{"name":"Test Service","url":"https://httpbin.org/status/200"}' \
		http://localhost:8080/api/add

# Удаление сервиса по индексу (например, индекс 0)
remove-service:
	curl -X POST -H "Content-Type: application/json" \
		-d '{"index":0}' \
		http://localhost:8080/api/remove
# Docker команды

# Сборка Docker образа
docker-build:
	@echo "Сборка Docker образа web-monitor..."
	docker build -t web-monitor:latest .

# Создание docker-compose.yml файла
docker-compose:
	@if [ -z "$(PORT)" ]; then echo "Ошибка: укажите PORT=номер_порта"; exit 1; fi
	@echo "Создание docker-compose.yml для порта $(PORT)..."
	@echo "version: '3.8'" > docker-compose.yml
	@echo "" >> docker-compose.yml
	@echo "services:" >> docker-compose.yml
	@echo "  web-monitor:" >> docker-compose.yml
	@echo "    image: web-monitor:latest" >> docker-compose.yml
	@echo "    container_name: web-monitor" >> docker-compose.yml
	@echo "    ports:" >> docker-compose.yml
	@echo "      - \"$(PORT):$(PORT)\"" >> docker-compose.yml
	@echo "    volumes:" >> docker-compose.yml
	@echo "      - ./data:/app/data" >> docker-compose.yml
	@echo "    command: [\"-port=$(PORT)\"]" >> docker-compose.yml
	@echo "    restart: unless-stopped" >> docker-compose.yml
	@echo "    environment:" >> docker-compose.yml
	@echo "      - TZ=Europe/Moscow" >> docker-compose.yml
	@echo "    networks:" >> docker-compose.yml
	@echo "      - web-monitor-network" >> docker-compose.yml
	@echo "" >> docker-compose.yml
	@echo "volumes:" >> docker-compose.yml
	@echo "  web-monitor-data:" >> docker-compose.yml
	@echo "    driver: local" >> docker-compose.yml
	@echo "" >> docker-compose.yml
	@echo "networks:" >> docker-compose.yml
	@echo "  web-monitor-network:" >> docker-compose.yml
	@echo "    driver: bridge" >> docker-compose.yml
	@echo "Файл docker-compose.yml создан для порта $(PORT)"
	@echo "Volume ./data будет использоваться для хранения services.json"

# Запуск через Docker Compose
docker-up:
	@if [ -z "$(PORT)" ]; then echo "Ошибка: укажите PORT=номер_порта"; exit 1; fi
	@if [ ! -f docker-compose.yml ]; then echo "Создание docker-compose.yml..."; make docker-compose PORT=$(PORT); fi
	@mkdir -p ./data
	@echo "Запуск web-monitor через Docker Compose на порту $(PORT)..."
	docker-compose up -d
	@echo "Сервис запущен на http://localhost:$(PORT)"
	@echo "Данные сохраняются в ./data/services.json"

# Остановка Docker Compose
docker-down:
	@echo "Остановка web-monitor..."
	docker-compose down
	@echo "Сервис остановлен"

# Полная очистка Docker (образы, контейнеры, volumes)
docker-clean:
	@echo "Остановка и удаление контейнеров..."
	-docker-compose down -v
	@echo "Удаление образа web-monitor..."
	-docker rmi web-monitor:latest
	@echo "Очистка завершена"