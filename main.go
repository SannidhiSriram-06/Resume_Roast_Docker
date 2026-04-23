package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqChoice struct {
	Message groqMessage `json:"message"`
}

type groqResponse struct {
	Choices []groqChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func extractPDFText(pdfPath string) (string, error) {
	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w (is poppler-utils installed?)", err)
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("PDF appears to be empty or image-only — try a text-based PDF")
	}
	return text, nil
}

func callGroq(apiKey, resumeText, jdText string) (string, error) {
	systemPrompt := `You are a brutally honest but constructive resume reviewer.
You roast the resume against the provided job description — no sugarcoating, no corporate fluff.
Your response must have exactly these four sections:

🔥 THE ROAST
A punchy, savage (but fair) roast of how the resume stacks up against the JD. Call out weak bullets, missing keywords, fluff, and mismatches. Be specific. Max 5 bullets.

📉 CRITICAL GAPS
Concrete skills/experience missing from the resume that the JD clearly demands. Max 5 bullets.

✅ WHAT'S ACTUALLY GOOD
Be honest — highlight what genuinely works. Don't fabricate. Max 3 bullets.

🛠️ FIX IT: TOP 5 ACTIONS
Specific, actionable improvements the candidate should make. Prioritized. Numbered 1–5.

Keep the tone sharp, smart, and direct — like a senior engineer giving real talk, not a recruiter.`

	userMsg := fmt.Sprintf("JOB DESCRIPTION:\n%s\n\n---\n\nRESUME:\n%s", jdText, resumeText)

	reqBody, _ := json.Marshal(groqRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
	})

	req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Groq request failed: %w", err)
	}
	defer resp.Body.Close()

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("failed to decode Groq response: %w", err)
	}
	if gr.Error != nil {
		return "", fmt.Errorf("Groq API error: %s", gr.Error.Message)
	}
	if len(gr.Choices) == 0 {
		return "", fmt.Errorf("Groq returned no choices")
	}
	return gr.Choices[0].Message.Content, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "resume-roaster"})
}

func roastHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		http.Error(w, `{"error":"GROQ_API_KEY not set"}`, http.StatusInternalServerError)
		return
	}

	r.ParseMultipartForm(10 << 20)

	jdText := strings.TrimSpace(r.FormValue("jd"))
	if jdText == "" {
		http.Error(w, `{"error":"Job description is required"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("resume")
	if err != nil {
		http.Error(w, `{"error":"Resume PDF is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		http.Error(w, `{"error":"Only PDF files are accepted"}`, http.StatusBadRequest)
		return
	}

	tmp, err := os.CreateTemp("", "resume-*.pdf")
	if err != nil {
		http.Error(w, `{"error":"Failed to save upload"}`, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		http.Error(w, `{"error":"Failed to write PDF"}`, http.StatusInternalServerError)
		return
	}
	tmp.Close()

	resumeText, err := extractPDFText(tmp.Name())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	roast, err := callGroq(apiKey, resumeText, jdText)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"roast": roast})
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join("static", filepath.Clean(r.URL.Path))
	if r.URL.Path == "/" {
		path = "static/index.html"
	}
	http.ServeFile(w, r, path)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/roast", roastHandler)
	mux.HandleFunc("/", staticHandler)

	log.Printf("Resume Roaster running on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
