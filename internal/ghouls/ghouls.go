package ghouls

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/toozej/ghouls/assets"
	"github.com/toozej/ghouls/templates"
)

type URLData struct {
	URLs []string `json:"urls"`
}

var (
	storageMutex sync.Mutex
	data         URLData
	dataFilePath string
)

func Serve() {
	// get data file path
	getDataFilePath()

	// Load URLs data file
	loadDataFromFile(dataFilePath)

	// Load the HTML template
	tmpl := template.Must(template.ParseFS(&templates.Templates, "*.html"))

	// Serve static files from the "static" directory
	setupStaticAssets()

	// Define a handler to render the template
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// You can pass data to the template if needed
		data := struct {
			URLs []string
		}{
			URLs: data.URLs, // Pass your list of URLs here
		}

		// Render the template with the data
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/add", addURL)
	http.HandleFunc("/delete", deleteURLs)
	http.HandleFunc("/list", listURLs)

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error listening & serving on port 8080 with 10sec timeout", err)
		return
	}
}

func getDataFilePath() {
	// Define the default path
	dataFilePath = "/data/data.json"

	// Check if the file exists at the "./data.json" path for local development
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory", err)
	}
	devDataFilePath := fmt.Sprintf("%s/data.json", cwd)
	if _, err := os.Stat(devDataFilePath); err == nil {
		dataFilePath = devDataFilePath
	}
}

func setupStaticAssets() {
	// serve regular static assets
	fs := &assets.Assets
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(fs))))
}

func addURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	url := r.FormValue("url")
	if url != "" {
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
		http.Error(w, "URL is missing", http.StatusBadRequest)
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

func saveDataToFile(dataFilePath string) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON in saveDataToFile()", err)
		return
	}
	if err := os.WriteFile(dataFilePath, dataJSON, 0600); err != nil {
		fmt.Println("Error saving data:", err)
		return
	}
}

func loadDataFromFile(dataFilePath string) {
	if _, err := os.Stat(dataFilePath); err == nil {
		dataJSON, err := os.ReadFile(dataFilePath) // #nosec G304
		if err != nil {
			fmt.Println("Error loading data JSON file in loadDataFromFile():", err)
			return
		}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			fmt.Println("Error unmarshalling JSON data in loadDataFromFile():", err)
			return
		}
	}
}
