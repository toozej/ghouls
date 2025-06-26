package ghouls

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/toozej/ghouls/templates"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Load the HTML template
	tmpl := template.Must(template.ParseFS(&templates.Templates, "*.html"))

	// Create template data with enhanced functionality
	templateData := struct {
		URLs      []string
		CsrfField template.HTML
		Stats     map[string]interface{}
	}{
		URLs:      data.URLs,
		CsrfField: csrf.TemplateField(r),
		Stats: map[string]interface{}{
			"TotalCount":  len(data.URLs),
			"DomainCount": countUniqueDomains(data.URLs),
		},
	}

	// Set content type for proper rendering
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render the template with the data
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte("OK")); err != nil { // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
		fmt.Println("Error writing health page", err)
		return
	}
}

func loginHandler(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suppliedUsername, suppliedPassword, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(suppliedUsername))
			passwordHash := sha256.Sum256([]byte(suppliedPassword))
			expectedUsernameHash := sha256.Sum256([]byte(username))
			expectedPasswordHash := sha256.Sum256([]byte(password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func addURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	rawURL := strings.TrimSpace(r.FormValue("url"))
	if rawURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Normalize and validate URL
	normalizedURL, err := normalizeURL(rawURL)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	storageMutex.Lock()
	defer storageMutex.Unlock()

	// Check if the URL already exists in the list
	for _, existingURL := range data.URLs {
		if existingURL == normalizedURL {
			// Instead of returning an error, redirect with a message
			// You could implement flash messages here if needed
			http.Redirect(w, r, "/?duplicate=true", http.StatusSeeOther)
			return
		}
	}

	// Add the URL to the beginning of the list (newest first)
	data.URLs = append([]string{normalizedURL}, data.URLs...)
	saveDataToFile(dataFilePath)

	// Redirect to the main page after a successful addition
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error parsing HTML form response", err)
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	selectedURLs := r.Form["urlsToDelete"]
	if len(selectedURLs) == 0 {
		http.Error(w, "No URLs selected for deletion", http.StatusBadRequest)
		return
	}

	storageMutex.Lock()
	defer storageMutex.Unlock()

	// Create a map for faster lookup
	urlsToDelete := make(map[string]bool)
	for _, url := range selectedURLs {
		urlsToDelete[url] = true
	}

	// Filter out URLs to delete
	var remainingURLs []string
	for _, url := range data.URLs {
		if !urlsToDelete[url] {
			remainingURLs = append(remainingURLs, url)
		}
	}

	data.URLs = remainingURLs
	saveDataToFile(dataFilePath)

	// Redirect to the main page after successful deletion
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func listURLs(w http.ResponseWriter, r *http.Request) {
	storageMutex.Lock()
	defer storageMutex.Unlock()

	// Enhanced JSON response with metadata
	response := map[string]interface{}{
		"urls":        data.URLs,
		"total_count": len(data.URLs),
		"domains":     getUniqueDomains(data.URLs),
	}

	urlsJSON, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshalling JSON in listURLs()", err)
		http.Error(w, "Error generating JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	if _, err := w.Write(urlsJSON); err != nil {
		fmt.Println("Error writing JSON response", err)
		return
	}
}

// Helper functions

// normalizeURL ensures the URL has a proper scheme and is valid
func normalizeURL(rawURL string) (string, error) {
	// Add https:// if no scheme is present
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Parse and validate the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Ensure we have a valid scheme and host
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: missing scheme or host")
	}

	// Return the normalized URL
	return parsedURL.String(), nil
}

// countUniqueDomains counts the number of unique domains in the URL list
func countUniqueDomains(urls []string) int {
	domains := make(map[string]bool)
	for _, rawURL := range urls {
		if parsedURL, err := url.Parse(rawURL); err == nil && parsedURL.Host != "" {
			domain := strings.ToLower(parsedURL.Host)
			// Remove www. prefix for counting
			domain = strings.TrimPrefix(domain, "www.")
			domains[domain] = true
		}
	}
	return len(domains)
}

// getUniqueDomains returns a list of unique domains from the URL list
func getUniqueDomains(urls []string) []string {
	domainMap := make(map[string]bool)
	for _, rawURL := range urls {
		if parsedURL, err := url.Parse(rawURL); err == nil && parsedURL.Host != "" {
			domain := strings.ToLower(parsedURL.Host)
			// Remove www. prefix
			domain = strings.TrimPrefix(domain, "www.")
			domainMap[domain] = true
		}
	}

	domains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		domains = append(domains, domain)
	}
	return domains
}
