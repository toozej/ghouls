package ghouls

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/toozej/ghouls/templates"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Load the HTML template
	tmpl := template.Must(template.ParseFS(&templates.Templates, "*.html"))

	// You can pass data to the template if needed
	data := struct {
		URLs      []string
		CsrfField template.HTML
	}{
		URLs:      data.URLs,             // Pass your list of URLs here
		CsrfField: csrf.TemplateField(r), // Pass the CSRF token here
	}

	// Render the template with the data
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
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

	url := r.FormValue("url")
	if url != "" && isValidURL(url) {
		storageMutex.Lock()
		defer storageMutex.Unlock()

		// Check if the URL already exists in the list
		for _, existingURL := range data.URLs {
			if existingURL == url {
				fmt.Fprintf(w, "URL already exists: %s", url) // nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
				return
			}
		}

		// If not a duplicate, add the URL
		data.URLs = append(data.URLs, url)
		saveDataToFile(dataFilePath)

		// Redirect to the main page after a successful addition
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else {
		http.Error(w, "URL is missing or invalid", http.StatusBadRequest)
		return
	}
}

func deleteURLs(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Println("Error parsing HTML form response", err)
		return
	}
	selectedURLs := r.Form["urlsToDelete"]

	if len(selectedURLs) == 0 {
		http.Error(w, "No URLs selected for deletion", http.StatusBadRequest)
		return
	}

	storageMutex.Lock()
	defer storageMutex.Unlock()

	for _, urlToDelete := range selectedURLs {
		for i, u := range data.URLs {
			if u == urlToDelete {
				data.URLs = append(data.URLs[:i], data.URLs[i+1:]...)
				break
			}
		}
	}

	saveDataToFile(dataFilePath)

	// Redirect to the main page after a successful deletion
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func listURLs(w http.ResponseWriter, r *http.Request) {
	storageMutex.Lock()
	defer storageMutex.Unlock()

	urlsJSON, err := json.Marshal(data.URLs)
	if err != nil {
		fmt.Println("Error marshalling JSON in listURLs()", err)
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(urlsJSON); err != nil { // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
		fmt.Println("Error parsing HTML form response", err)
		return
	}
}
