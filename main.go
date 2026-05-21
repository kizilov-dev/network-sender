package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	r.GET("/", handleRoot)
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/send", handleSend)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("[STARTUP] Network Sender запущен на %s", addr)
	log.Printf("[STARTUP] Использование: GET /send?target=http://HOST:PORT/path")
	if err := r.Run(addr); err != nil {
		log.Fatalf("[STARTUP] Ошибка запуска сервера: %v", err)
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		log.Printf("[REQUEST] --> %s %s от %s | User-Agent: %s",
			c.Request.Method, c.Request.URL.String(), c.ClientIP(), c.Request.UserAgent())

		c.Next()

		elapsed := time.Since(start)
		log.Printf("[REQUEST] <-- %s %s | Статус: %d | Время: %s",
			c.Request.Method, c.Request.URL.String(), c.Writer.Status(), elapsed)
	}
}

func handleRoot(c *gin.Context) {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	exampleURL := fmt.Sprintf("%s://%s/send?target=http://192.168.0.10:8081/ping", scheme, c.Request.Host)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Network Sender</title></head>
<body>
<h1>Network Sender</h1>
<p>Отправляет HTTP-запрос на указанный адрес в локальной сети и показывает результат.</p>
<h3>Пример запроса:</h3>
<pre><a href="%s">%s</a></pre>
<h3>Параметры:</h3>
<ul>
  <li><b>target</b> — полный URL для запроса (обязательный), например: <code>http://192.168.0.10:8081/ping</code></li>
  <li><b>timeout</b> — таймаут в секундах (необязательный, по умолчанию 10)</li>
</ul>
</body>
</html>`, exampleURL, exampleURL)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func handleSend(c *gin.Context) {
	target := c.Query("target")
	if target == "" {
		log.Printf("[SEND] Ошибка: параметр 'target' не указан")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Обязательный параметр: target. Пример: /send?target=http://192.168.0.10:8081/ping",
		})
		return
	}

	timeoutStr := c.DefaultQuery("timeout", "10")
	var timeoutSec int
	if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutSec); err != nil || timeoutSec < 1 || timeoutSec > 60 {
		timeoutSec = 10
	}
	timeout := time.Duration(timeoutSec) * time.Second

	log.Printf("[SEND] ======================================")
	log.Printf("[SEND] Начинаю отправку запроса")
	log.Printf("[SEND]   Target: %s", target)
	log.Printf("[SEND]   Timeout: %s", timeout)
	log.Printf("[SEND] ======================================")

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		log.Printf("[SEND] ОШИБКА создания запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"target":  target,
			"success": false,
			"error":   fmt.Sprintf("Невалидный URL: %v", err),
		})
		return
	}
	req.Header.Set("X-Sender-App", "network-sender")
	req.Header.Set("X-Request-Time", time.Now().Format(time.RFC3339Nano))

	log.Printf("[SEND] Отправляю GET %s ...", target)
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[SEND] ❌ ОШИБКА запроса к %s", target)
		log.Printf("[SEND]   Ошибка: %v", err)
		log.Printf("[SEND]   Время до ошибки: %s", elapsed)
		log.Printf("[SEND] ======================================")

		c.JSON(http.StatusOK, gin.H{
			"target":  target,
			"success": false,
			"latency": elapsed.String(),
			"error":   err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	bodyStr := ""
	if readErr != nil {
		log.Printf("[SEND] Предупреждение: ошибка чтения тела ответа: %v", readErr)
		bodyStr = fmt.Sprintf("<ошибка чтения: %v>", readErr)
	} else {
		bodyStr = string(body)
		if len(bodyStr) > 1000 {
			bodyStr = bodyStr[:1000] + "... (обрезано)"
		}
	}

	log.Printf("[SEND] ✅ Получен ответ от %s", target)
	log.Printf("[SEND]   HTTP статус: %d %s", resp.StatusCode, resp.Status)
	log.Printf("[SEND]   Время ответа: %s", elapsed)
	log.Printf("[SEND]   Content-Type: %s", resp.Header.Get("Content-Type"))
	log.Printf("[SEND]   Content-Length: %s", resp.Header.Get("Content-Length"))
	log.Printf("[SEND]   Тело ответа: %s", bodyStr)
	log.Printf("[SEND] ======================================")

	c.JSON(http.StatusOK, gin.H{
		"target":         target,
		"success":        true,
		"latency":        elapsed.String(),
		"status_code":    resp.StatusCode,
		"response_body":  bodyStr,
		"content_type":   resp.Header.Get("Content-Type"),
		"content_length": resp.Header.Get("Content-Length"),
	})
}
