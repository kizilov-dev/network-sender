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
	host := c.Request.Host

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Network Sender</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; padding: 20px; }
  .container { max-width: 800px; margin: 0 auto; }
  h1 { margin-bottom: 8px; }
  .subtitle { color: #666; margin-bottom: 24px; }
  .card { background: #fff; border-radius: 8px; padding: 24px; margin-bottom: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  .card h2 { margin-bottom: 16px; font-size: 18px; }
  .form-row { display: flex; gap: 12px; margin-bottom: 12px; align-items: flex-end; flex-wrap: wrap; }
  .form-group { display: flex; flex-direction: column; }
  .form-group label { font-size: 13px; font-weight: 600; margin-bottom: 4px; color: #555; }
  .form-group input { padding: 8px 12px; border: 1px solid #ccc; border-radius: 4px; font-size: 14px; }
  .form-group input:focus { outline: none; border-color: #4a90d9; box-shadow: 0 0 0 2px rgba(74,144,217,0.2); }
  .form-group.target { flex: 1; min-width: 200px; }
  .form-group.timeout { width: 100px; }
  button { padding: 8px 20px; background: #4a90d9; color: #fff; border: none; border-radius: 4px; font-size: 14px; cursor: pointer; font-weight: 600; height: 36px; }
  button:hover { background: #3a7bc8; }
  button:disabled { background: #aaa; cursor: wait; }
  #result { display: none; margin-top: 16px; }
  #result.visible { display: block; }
  .result-header { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
  .badge { padding: 3px 10px; border-radius: 12px; font-size: 13px; font-weight: 600; color: #fff; }
  .badge.success { background: #34a853; }
  .badge.error { background: #ea4335; }
  .result-meta { font-size: 13px; color: #666; }
  .result-body { background: #f8f8f8; border: 1px solid #e0e0e0; border-radius: 4px; padding: 12px; font-family: monospace; font-size: 13px; white-space: pre-wrap; word-break: break-all; max-height: 300px; overflow-y: auto; }
  .steps { counter-reset: step; list-style: none; padding: 0; }
  .steps li { counter-increment: step; padding: 8px 0 8px 36px; position: relative; border-bottom: 1px solid #eee; }
  .steps li:last-child { border-bottom: none; }
  .steps li::before { content: counter(step); position: absolute; left: 0; top: 8px; width: 24px; height: 24px; background: #4a90d9; color: #fff; border-radius: 50%%; font-size: 13px; font-weight: 600; display: flex; align-items: center; justify-content: center; }
  code { background: #eee; padding: 2px 6px; border-radius: 3px; font-size: 13px; }
  .endpoints { margin: 8px 0; }
  .endpoints dt { font-weight: 600; margin-top: 8px; font-family: monospace; }
  .endpoints dd { margin-left: 16px; color: #555; font-size: 14px; }
  .note { background: #fff8e1; border-left: 3px solid #ffc107; padding: 12px 16px; border-radius: 0 4px 4px 0; margin-top: 12px; font-size: 14px; }
</style>
</head>
<body>
<div class="container">
  <h1>Network Sender</h1>
  <p class="subtitle">Тестирование HTTP-связности между сервисами в локальной сети облачного провайдера</p>

  <div class="card">
    <h2>Отправить запрос</h2>
    <form id="sendForm">
      <div class="form-row">
        <div class="form-group target">
          <label for="target">Target URL</label>
          <input type="text" id="target" placeholder="http://192.168.0.10:8081/ping" required>
        </div>
        <div class="form-group timeout">
          <label for="timeout">Timeout (сек)</label>
          <input type="number" id="timeout" value="10" min="1" max="60">
        </div>
        <button type="submit" id="sendBtn">Отправить</button>
      </div>
    </form>
    <div id="result">
      <div class="result-header">
        <span id="resultBadge" class="badge"></span>
        <span id="resultMeta" class="result-meta"></span>
      </div>
      <div id="resultBody" class="result-body"></div>
    </div>
  </div>

  <div class="card">
    <h2>Как использовать</h2>
    <ol class="steps">
      <li>
        Запустите <b>Network Receiver</b> на целевой машине:
        <br><code>PORT=8081 go run main.go</code>
      </li>
      <li>
        Запустите <b>Network Sender</b> (это приложение) на другой машине:
        <br><code>PORT=8080 go run main.go</code>
      </li>
      <li>
        Откройте эту страницу в браузере и введите локальный адрес receiver в поле <b>Target URL</b>,
        например: <code>http://192.168.0.10:8081/ping</code>
      </li>
      <li>
        Нажмите <b>Отправить</b> — sender сделает HTTP-запрос к receiver по локальной сети.
        Результат появится на этой странице.
      </li>
      <li>
        Проверьте логи обоих приложений в терминале — там будет подробная информация о запросе.
      </li>
    </ol>
    <div class="note">
      Можно также использовать API напрямую:
      <code>curl "%s://%s/send?target=http://RECEIVER_IP:8081/ping"</code>
    </div>
  </div>

  <div class="card">
    <h2>Эндпоинты Receiver</h2>
    <p>Приложение Network Receiver предоставляет следующие эндпоинты, на которые можно отправлять запросы:</p>
    <dl class="endpoints">
      <dt>GET /ping</dt>
      <dd>Простая проверка связи. Возвращает <code>{"message":"pong"}</code> с таймстемпом и IP клиента.</dd>
      <dt>ANY /echo/*</dt>
      <dd>Эхо-эндпоинт. Принимает любой HTTP-метод, возвращает все детали запроса: метод, путь, заголовки, query-параметры.</dd>
      <dt>GET /healthz</dt>
      <dd>Health-check. Возвращает <code>ok</code>.</dd>
    </dl>
  </div>

  <div class="card">
    <h2>Что смотреть в логах</h2>
    <dl class="endpoints">
      <dt>Sender (это приложение)</dt>
      <dd>Теги <code>[SEND]</code> — целевой URL, таймаут, HTTP-статус ответа, тело ответа, время запроса. При ошибке — текст ошибки и время до ошибки.</dd>
      <dt>Receiver (удалённое приложение)</dt>
      <dd>Теги <code>[INCOMING]</code> — метод, путь, IP клиента, Remote Addr, все HTTP-заголовки, время обработки. Теги <code>[PING]</code> / <code>[ECHO]</code> — детали конкретного эндпоинта.</dd>
    </dl>
  </div>
</div>

<script>
const form = document.getElementById("sendForm");
const btn = document.getElementById("sendBtn");
const resultDiv = document.getElementById("result");
const badge = document.getElementById("resultBadge");
const meta = document.getElementById("resultMeta");
const body = document.getElementById("resultBody");

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const target = document.getElementById("target").value.trim();
  const timeout = document.getElementById("timeout").value || "10";
  if (!target) return;

  btn.disabled = true;
  btn.textContent = "Отправка...";
  resultDiv.className = "";

  try {
    const url = "/send?target=" + encodeURIComponent(target) + "&timeout=" + encodeURIComponent(timeout);
    const resp = await fetch(url);
    const data = await resp.json();

    resultDiv.className = "visible";
    if (data.success) {
      badge.className = "badge success";
      badge.textContent = "OK";
      meta.textContent = "HTTP " + data.status_code + " | " + data.latency + " | " + (data.content_type || "");
      body.textContent = data.response_body;
    } else {
      badge.className = "badge error";
      badge.textContent = "FAIL";
      meta.textContent = data.latency ? ("Время: " + data.latency) : "";
      body.textContent = data.error || JSON.stringify(data, null, 2);
    }
  } catch (err) {
    resultDiv.className = "visible";
    badge.className = "badge error";
    badge.textContent = "ERROR";
    meta.textContent = "";
    body.textContent = err.toString();
  } finally {
    btn.disabled = false;
    btn.textContent = "Отправить";
  }
});
</script>
</body>
</html>`, scheme, host)

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
