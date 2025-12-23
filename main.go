package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Service struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status bool   `json:"status"`
}

type Monitor struct {
	services []Service
	mutex    sync.RWMutex
	filename string
}

func NewMonitor(filename string) *Monitor {
	return &Monitor{
		services: make([]Service, 0),
		filename: filename,
	}
}

func (m *Monitor) AddService(name, url string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	service := Service{
		Name:   name,
		URL:    url,
		Status: false,
	}
	m.services = append(m.services, service)
	m.saveToFile()
}

func (m *Monitor) RemoveService(index int) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if index < 0 || index >= len(m.services) {
		return false
	}
	
	// Удаляем элемент из слайса
	m.services = append(m.services[:index], m.services[index+1:]...)
	m.saveToFile()
	return true
}

func (m *Monitor) LoadFromFile() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Проверяем существование файла
	if _, err := os.Stat(m.filename); os.IsNotExist(err) {
		fmt.Printf("Файл %s не найден, создаем новый список сервисов\n", m.filename)
		return nil
	}
	
	// Читаем файл
	data, err := ioutil.ReadFile(m.filename)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла %s: %v", m.filename, err)
	}
	
	// Парсим JSON
	if err := json.Unmarshal(data, &m.services); err != nil {
		return fmt.Errorf("ошибка парсинга JSON из файла %s: %v", m.filename, err)
	}
	
	fmt.Printf("Загружено %d сервисов из файла %s\n", len(m.services), m.filename)
	return nil
}

func (m *Monitor) saveToFile() error {
	// Сериализуем в JSON
	data, err := json.MarshalIndent(m.services, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации в JSON: %v", err)
	}
	
	// Записываем в файл
	if err := ioutil.WriteFile(m.filename, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи в файл %s: %v", m.filename, err)
	}
	
	return nil
}

func (m *Monitor) GetServices() []Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	services := make([]Service, len(m.services))
	copy(services, m.services)
	return services
}

func (m *Monitor) CheckService(url string) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

func (m *Monitor) CheckAllServices() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for i := range m.services {
		status := m.CheckService(m.services[i].URL)
		m.services[i].Status = status
	}
}

func getServicesFilePath() string {
	// Проверяем, запущены ли мы в Docker (наличие папки /app/data)
	if _, err := os.Stat("/app/data"); err == nil {
		return "/app/data/services.json"
	}
	// Иначе используем текущую директорию
	return "services.json"
}

var monitor *Monitor

func main() {
	// Определяем флаг для порта
	port := flag.String("port", "", "Порт для запуска сервера (обязательный параметр)")
	flag.Parse()
	
	// Проверяем, что порт указан
	if *port == "" {
		fmt.Println("Ошибка: необходимо указать порт через флаг -port")
		fmt.Println("Пример: go run main.go -port=8080")
		return
	}
	
	// Инициализируем монитор с файлом для сохранения
	servicesFile := getServicesFilePath()
	monitor = NewMonitor(servicesFile)
	
	// Создаем директорию для данных если нужно
	if dir := filepath.Dir(servicesFile); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Ошибка создания директории %s: %v", dir, err)
		}
	}
	
	// Загружаем сервисы из файла
	if err := monitor.LoadFromFile(); err != nil {
		log.Printf("Ошибка загрузки сервисов: %v", err)
	}
	
	// Если файл не существовал или был пуст, добавляем тестовые сервисы
	if len(monitor.GetServices()) == 0 {
		fmt.Println("Добавляем тестовые сервисы...")
		monitor.AddService("Google", "https://www.google.com")
		monitor.AddService("GitHub", "https://github.com")
	}
	
	// Настраиваем маршруты
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/services", servicesHandler)
	http.HandleFunc("/api/add", addServiceHandler)
	http.HandleFunc("/api/remove", removeServiceHandler)
	
	addr := ":" + *port
	fmt.Printf("Сервер запущен на http://localhost:%s\n", *port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Мониторинг веб-сервисов</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            text-align: center;
        }
        .service-list {
            margin: 20px 0;
        }
        .service-item {
            display: flex;
            align-items: center;
            padding: 10px;
            margin: 5px 0;
            background: #f9f9f9;
            border-radius: 4px;
            border-left: 4px solid #ddd;
        }
        .service-item.offline {
            animation: blink-red 2s infinite;
        }
        @keyframes blink-red {
            0%, 50% {
                background-color: #f9f9f9;
            }
            25%, 75% {
                background-color: #f44336;
            }
        }
        .service-info {
            flex: 1;
            display: flex;
            align-items: center;
        }
        .delete-btn {
            background: #dc3545;
            color: white;
            border: none;
            border-radius: 50%;
            width: 24px;
            height: 24px;
            cursor: pointer;
            font-size: 14px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-left: 10px;
        }
        .delete-btn:hover {
            background: #c82333;
        }
        .status-light {
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 10px;
        }
        .status-online {
            background-color: #4CAF50;
            box-shadow: 0 0 6px #4CAF50;
        }
        .status-offline {
            background-color: #f44336;
            box-shadow: 0 0 6px #f44336;
        }
        .service-name {
            font-weight: bold;
            margin-right: 10px;
        }
        .add-form {
            margin-top: 30px;
            padding: 20px;
            background: #f0f0f0;
            border-radius: 4px;
        }
        .form-group {
            margin: 10px 0;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input[type="text"], input[type="url"] {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        button {
            background: #007cba;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: #005a87;
        }
        .refresh-controls {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .refresh-btn {
            background: #28a745;
        }
        .refresh-btn:hover {
            background: #218838;
        }
        .countdown {
            font-size: 0.9em;
            color: #666;
            font-weight: normal;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Мониторинг веб-сервисов</h1>
        
        <div class="refresh-controls">
            <div class="countdown">
                Следующее обновление через: <span id="countdown">10</span> сек
            </div>
            <button class="refresh-btn" onclick="manualRefresh()">Обновить сейчас</button>
        </div>
        
        <div class="service-list" id="serviceList">
            <p>Загрузка сервисов...</p>
        </div>
        
        <div class="add-form">
            <h3>Добавить новый сервис</h3>
            <form id="addServiceForm">
                <div class="form-group">
                    <label for="serviceName">Название сервиса:</label>
                    <input type="text" id="serviceName" name="name" required>
                </div>
                <div class="form-group">
                    <label for="serviceUrl">URL сервиса:</label>
                    <input type="url" id="serviceUrl" name="url" required placeholder="https://example.com">
                </div>
                <button type="submit">Добавить сервис</button>
            </form>
        </div>
    </div>

    <script>
        let countdownTimer;
        let refreshTimer;
        let countdownValue = 10;

        function updateCountdown() {
            document.getElementById('countdown').textContent = countdownValue;
            if (countdownValue <= 0) {
                countdownValue = 10;
                loadServices();
            } else {
                countdownValue--;
            }
        }

        function startCountdown() {
            countdownValue = 10;
            clearInterval(countdownTimer);
            clearInterval(refreshTimer);
            
            countdownTimer = setInterval(updateCountdown, 1000);
            refreshTimer = setInterval(function() {
                loadServices();
            }, 10000);
        }

        function manualRefresh() {
            loadServices();
            startCountdown(); // Перезапускаем счетчик
        }

        function loadServices() {
            fetch('/api/services')
                .then(response => response.json())
                .then(services => {
                    const serviceList = document.getElementById('serviceList');
                    if (services.length === 0) {
                        serviceList.innerHTML = '<p>Нет добавленных сервисов</p>';
                        return;
                    }
                    
                    serviceList.innerHTML = services.map((service, index) => 
                        '<div class="service-item' + (service.status ? '' : ' offline') + '">' +
                            '<div class="service-info">' +
                                '<div class="status-light ' + (service.status ? 'status-online' : 'status-offline') + '"></div>' +
                                '<span class="service-name">' + service.name + '</span>' +
                            '</div>' +
                            '<button class="delete-btn" onclick="removeService(' + index + ')" title="Удалить сервис">×</button>' +
                        '</div>'
                    ).join('');
                })
                .catch(error => {
                    console.error('Ошибка загрузки сервисов:', error);
                    document.getElementById('serviceList').innerHTML = '<p>Ошибка загрузки сервисов</p>';
                });
        }

        function removeService(index) {
            if (confirm('Вы уверены, что хотите удалить этот сервис?')) {
                fetch('/api/remove', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({index: index})
                })
                .then(response => response.json())
                .then(result => {
                    if (result.success) {
                        loadServices();
                    } else {
                        alert('Ошибка удаления сервиса: ' + result.error);
                    }
                })
                .catch(error => {
                    console.error('Ошибка:', error);
                    alert('Ошибка удаления сервиса');
                });
            }
        }

        document.getElementById('addServiceForm').addEventListener('submit', function(e) {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const data = {
                name: formData.get('name'),
                url: formData.get('url')
            };
            
            fetch('/api/add', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            })
            .then(response => response.json())
            .then(result => {
                if (result.success) {
                    e.target.reset();
                    loadServices();
                } else {
                    alert('Ошибка добавления сервиса: ' + result.error);
                }
            })
            .catch(error => {
                console.error('Ошибка:', error);
                alert('Ошибка добавления сервиса');
            });
        });

        // Загружаем сервисы при загрузке страницы
        loadServices();
        
        // Запускаем счетчик
        startCountdown();
    </script>
</body>
</html>
	`
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, tmpl)
}

func servicesHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем все сервисы в момент запроса
	monitor.CheckAllServices()
	
	w.Header().Set("Content-Type", "application/json")
	services := monitor.GetServices()
	json.NewEncoder(w).Encode(services)
}

func addServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Неверный формат данных",
		})
		return
	}
	
	if req.Name == "" || req.URL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Название и URL обязательны",
		})
		return
	}
	
	monitor.AddService(req.Name, req.URL)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
func removeServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Index int `json:"index"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Неверный формат данных",
		})
		return
	}
	
	if !monitor.RemoveService(req.Index) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Неверный индекс сервиса",
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}