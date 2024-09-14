package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var logs []string

func main() {
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
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
        background-color: #d4edda;
        color: #155724;
        padding: 10px;
        border: 1px solid #c3e6cb;
        border-radius: 5px;
        margin-bottom: 10px;
      }
      .status-error {
        background-color: #f8d7da;
        color: #721c24;
        padding: 10px;
        border: 1px solid #f5c6cb;
        border-radius: 5px;
        margin-bottom: 10px;
      }
      .output {
        background-color: #28a745;
        padding: 10px;
        margin-top: 5px;
        margin-bottom: 5px;
        display: inline-block;
        color: white;
        border-radius: 5px;
        white-space: pre-wrap;
      }
      pre {
        white-space: pre-wrap;
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
        <h6>URL</h6>
        <form action="/run" method="post" enctype="multipart/form-data">
            <input type="text" name="url" placeholder="https://example.com/home.php?id={payload}" aria-label="Text">
            <br><br>
            <label>Upload payload file:</label>
            <input type="file" name="payload-file">
            <button type="submit" class="pico-background-slate-900">Run</button>
        </form>
    </main>

    <article>
      <header>
        <p><strong>Results</strong></p>
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

		// Process the uploaded file
		file, _, err := r.FormFile("payload-file")
		if err != nil {
			http.Error(w, "Error reading file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read payloads from the file
		var payloads []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			payloads = append(payloads, scanner.Text())
		}

		if inputURL == "" || len(payloads) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var results []string
		for _, payload := range payloads {
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
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			if resp.StatusCode == http.StatusOK {
				if strings.Contains(bodyStr, "alert") {
					statusClass = "output"
					statusOutput = fmt.Sprintf("<div class='%s'>XSS Injected Successfully!</div>", statusClass)
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
