package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/mattn/go-sqlite3"
)

// Структура конфигурации
type Config struct {
	DNSPort int
	APIPort int
	JWTSecret string
}

// Структура перенаправления
type RedirectRule struct {
	Domain string
	DNS    string // Формат: "1.1.1.1:853" (поддержка DoT)
}

// Глобальные переменные
var (
	config         Config
	blockedDomains = make(map[string]bool)
	redirectRules  = make(map[string]string)
	db             *System: You are Grok 3 built by xAI. db             *sql.DB
	mu             sync.RWMutex
	// Prometheus метрики
	totalRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dns_requests_total",
		Help: "Total number of DNS requests",
	})
	blockedRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dns_blocked_requests_total",
		Help: "Total number of blocked DNS requests",
	})
)

// Инициализация базы данных
func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./analytics.db")
	if err != nil {
		log.Fatalf("Ошибка подключения к SQLite: %v", err)
	}

	// Таблица аналитики
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS analytics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME,
			domain TEXT,
			blocked BOOLEAN,
			type TEXT
		)
	`)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы analytics: %v", err)
	}

	// Таблица блоклистов
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS blocklists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE
		)
	`)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы blocklists: %v", err)
	}

	// Таблица перенаправлений
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS redirects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT UNIQUE,
			dns TEXT
		)
	`)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы redirects: %v", err)
	}

	// Таблица конфигурации
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы config: %v", err)
	}

	// Инициализация конфигурации
	_, err = db.Exec(`
		INSERT OR IGNORE INTO config (key, value) VALUES
		('dns_port', '53'),
		('api_port', '8080'),
		('jwt_secret', 'your_jwt_secret_here')
	`)
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}
}

// Загрузка конфигурации
func loadConfig() {
	row := db.QueryRow("SELECT value FROM config WHERE key = 'dns_port'")
	if err := row.Scan(&config.DNSPort); err != nil {
		log.Fatalf("Ошибка загрузки dns_port: %v", err)
	}
	row = db.QueryRow("SELECT value FROM config WHERE key = 'api_port'")
	if err := row.Scan(&config.APIPort); err != nil {
		log.Fatalf("Ошибка загрузки api_port: %v", err)
	}
	row = db.QueryRow("SELECT value FROM config WHERE key = 'jwt_secret'")
	if err := row.Scan(&config.JWTSecret); err != nil {
		log.Fatalf("Ошибка загрузки jwt_secret: %v", err)
	}
}

// Загрузка блоклистов
func loadBlocklists() {
	rows, err := db.Query("SELECT url FROM blocklists")
	if err != nil {
		log.Printf("Ошибка загрузки блоклистов: %v", err)
		return
	}
	defer rows.Close()

	mu.Lock()
	blockedDomains = make(map[string]bool)
	for rows.Next() {
		var url string
		rows.Scan(&url)
		data, err := os.ReadFile(url) // Для простоты читаем локальный файл
		if err != nil {
			log.Printf("Ошибка загрузки блоклиста %s: %v", url, err)
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				parts := strings.Fields(line)
				domain := parts[len(parts)-1]
				blockedDomains[domain] = true
			}
		}
	}
	mu.Unlock()
	log.Printf("Загружено %d доменов в блоклист", len(blockedDomains))
}

// Загрузка правил перенаправления
func loadRedirectRules() {
	rows, err := db.Query("SELECT domain, dns FROM redirects")
	if err != nil {
		log.Printf("Ошибка загрузки правил перенаправления: %v", err)
		return
	}
	defer rows.Close()

	mu.Lock()
	redirectRules = make(map[string]string)
	for rows.Next() {
		var domain, dns string
		rows.Scan(&domain, &dns)
		redirectRules[domain] = dns
	}
	mu.Unlock()
	log.Printf("Загружено %d правил перенаправления", len(redirectRules))
}

// Сохранение аналитики
func saveAnalytics(domain, qtype string, blocked bool) {
	mu.Lock()
	defer mu.Unlock()
	_, err := db.Exec(
		"INSERT INTO analytics (timestamp, domain, blocked, type) VALUES (?, ?, ?, ?)",
		time.Now(), domain, blocked, qtype,
	)
	if err != nil {
		log.Printf("Ошибка сохранения аналитики: %v", err)
	}
}

// Обработчик DNS-запросов
func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	domain := strings.TrimSuffix(r.Question[0].Name, ".")
	qtype := dns.TypeToString[r.Question[0].Qtype]

	totalRequests.Inc()
	saveAnalytics(domain, qtype, blockedDomains[domain])

	mu.RLock()
	if blockedDomains[domain] {
		mu.RUnlock()
		blockedRequests.Inc()
		msg.SetRcode(r, dns.RcodeRefused)
		w.WriteMsg(&msg)
		return
	}

	// Проверка перенаправления
	if dnsAddr, ok := redirectRules[domain]; ok {
		mu.RUnlock()
		// Предполагаем, что dnsAddr в формате "ip:port" (поддержка DoT)
		client := &dns.Client{
			Net: "tcp-tls",
			TLSConfig: &tls.Config{
				ServerName: strings.Split(dnsAddr, ":")[0],
			},
		}
		resp, _, err := client.Exchange(r, dnsAddr)
		if err != nil {
			log.Printf("Ошибка перенаправления на %s: %v", dnsAddr, err)
			msg.SetRcode(r, dns.RcodeServerFailure)
		} else {
			msg.Answer = resp.Answer
		}
	} else {
		mu.RUnlock()
		// Перенаправление к Cloudflare DNS-over-TLS
		client := &dns.Client{
			Net: "tcp-tls",
			TLSConfig: &tls.Config{
				ServerName: "frd4wvnobp.cloudflare-gateway.com",
			},
		}
		resp, _, err := client.Exchange(r, "1.1.1.1:853")
		if err != nil {
			log.Printf("Ошибка DoT перенаправления: %v", err)
			msg.SetRcode(r, dns.RcodeServerFailure)
		} else {
			msg.Answer = resp.Answer
		}
	}
	w.WriteMsg(&msg)
}

// JWT middleware
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(401, gin.H{"error": "Требуется авторизация"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Недействительный токен"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// API
func setupAPI() *gin.Engine {
	r := gin.Default()

	// Аутентификация
	r.POST("/login", func(c *gin.Context) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&creds); err != nil {
			c.JSON(400, gin.H{"error": "Неверный запрос"})
			return
		}
		if creds.Username == "admin" && creds.Password == "password" {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"username": creds.Username,
				"exp":      time.Now().Add(time.Hour * 24).Unix(),
			})
			tokenString, _ := token.SignedString([]byte(config.JWTSecret))
			c.JSON(200, gin.H{"token": tokenString})
		} else {
			c.JSON(401, gin.H{"error": "Неверные данные"})
		}
	})

	// Защищённые роуты
	protected := r.Group("/api", authMiddleware())

	// Статистика
	protected.GET("/stats", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT COUNT(*) as total, SUM(CASE WHEN blocked THEN 1 ELSE 0 END) as blocked
			FROM analytics
		`)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка базы данных"})
			return
		}
		defer rows.Close()
		var total, blocked int
		if rows.Next() {
			rows.Scan(&total, &blocked)
		}

		rows, err = db.Query(`
			SELECT domain, COUNT(*) as count
			FROM analytics
			GROUP BY domain
			ORDER BY count DESC
			LIMIT 5
		`)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка базы данных"})
			return
		}
		defer rows.Close()
		topDomains := make(map[string]int)
		for rows.Next() {
			var domain string
			var count int
			rows.Scan(&domain, &count)
			topDomains[domain] = count
		}

		c.JSON(200, gin.H{
			"total_requests": total,
			"blocked":        blocked,
			"top_domains":    topDomains,
		})
	})

	// QPS
	protected.GET("/qps", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT strftime('%Y-%m-%dT%H:%M', timestamp) as minute, COUNT(*) as count
			FROM analytics
			WHERE timestamp >= datetime('now', '-24 hours')
			GROUP BY minute
		`)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка базы данных"})
			return
		}
		defer rows.Close()
		qps := make(map[string]int)
		for rows.Next() {
			var minute string
			var count int
			rows.Scan(&minute, &count)
			qps[minute] = count
		}
		c.JSON(200, qps)
	})

	// Добавление домена в блоклист
	protected.POST("/block", func(c *gin.Context) {
		var req struct {
			Domain string `json:"domain"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Неверный запрос"})
			return
		}
		mu.Lock()
		blockedDomains[req.Domain] = true
		mu.Unlock()
		c.JSON(200, gin.H{"message": "Домен добавлен в блоклист"})
	})

	// Управление блоклистами
	protected.GET("/blocklists", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, url FROM blocklists")
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка базы данных"})
			return
		}
		defer rows.Close()
		var blocklists []map[string]interface{}
		for rows.Next() {
			var id int
			var url string
			rows.Scan(&id, &url)
			blocklists = append(blocklists, map[string]interface{}{"id": id, "url": url})
		}
		c.JSON(200, blocklists)
	})

	protected.POST("/blocklists", func(c *gin.Context) {
		var req struct {
			URL string `json:"url"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Неверный запрос"})
			return
		}
		_, err := db.Exec("INSERT INTO blocklists (url) VALUES (?)", req.URL)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка добавления блоклиста"})
			return
		}
		loadBlocklists()
		c.JSON(200, gin.H{"message": "Блоклист добавлен"})
	})

	protected.DELETE("/blocklists/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM blocklists WHERE id = ?", id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка удаления блоклиста"})
			return
		}
		loadBlocklists()
		c.JSON(200, gin.H{"message": "Блоклист удалён"})
	})

	// Управление перенаправлениями
	protected.GET("/redirects", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, domain, dns FROM redirects")
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка базы данных"})
			return
		}
		defer rows.Close()
		var redirects []map[string]interface{}
		for rows.Next() {
			var id int
			var domain, dns string
			rows.Scan(&id, &domain, &dns)
			redirects = append(redirects, map[string]interface{}{"id": id, "domain": domain, "dns": dns})
		}
		c.JSON(200, redirects)
	})

	protected.POST("/redirects", func(c *gin.Context) {
		var req struct {
			Domain string `json:"domain"`
			DNS    string `json:"dns"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Неверный запрос"})
			return
		}
		_, err := db.Exec("INSERT INTO redirects (domain, dns) VALUES (?, ?)", req.Domain, req.DNS)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка добавления перенаправления"})
			return
		}
		loadRedirectRules()
		c.JSON(200, gin.H{"message": "Перенаправление добавлено"})
	})

	protected.DELETE("/redirects/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM redirects WHERE id = ?", id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка удаления перенаправления"})
			return
		}
		loadRedirectRules()
		c.JSON(200, gin.H{"message": "Перенаправление удалено"})
	})

	// Управление конфигурацией
	protected.GET("/config", func(c *gin.Context) {
		rows, err := db.Query("SELECT key, value FROM config")
		if err != nil {
			c.JSON(500, gin.H{"error": "Ошибка базы данных"})
			return
		}
		defer rows.Close()
		configMap := make(map[string]string)
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			configMap[key] = value
		}
		c.JSON(200, configMap)
	})

	protected.PUT("/config", func(c *gin.Context) {
		var req map[string]string
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Неверный запрос"})
			return
		}
		for key, value := range req {
			_, err := db.Exec("UPDATE config SET value = ? WHERE key = ?", value, key)
			if err != nil {
				c.JSON(500, gin.H{"error": fmt.Sprintf("Ошибка обновления %s", key)})
				return
			}
		}
		loadConfig()
		c.JSON(200, gin.H{"message": "Конфигурация обновлена"})
	})

	// Prometheus метрики
	r.GET("/metrics", gin.HandlerFunc(promhttp.Handler().ServeHTTP))

	return r
}

func main() {
	// Prometheus
	prometheus.MustRegister(totalRequests, blockedRequests)

	// Инициализация
	initDB()
	defer db.Close()
	loadConfig()
	loadBlocklists()
	loadRedirectRules()

	// DNS-сервер
	dns.HandleFunc(".", handleDNSRequest)
	server := &dns.Server{Addr: fmt.Sprintf(":%d", config.DNSPort), Net: "udp"}
	go func() {
		log.Printf("Запуск DNS-сервера на порту %d", config.DNSPort)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Ошибка DNS-сервера: %v", err)
		}
	}()

	// API
	r := setupAPI()
	log.Printf("Запуск API на порту %d", config.APIPort)
	if err := r.Run(fmt.Sprintf(":%d", config.APIPort)); err != nil {
		log.Fatalf("Ошибка API: %v", err)
	}
}
