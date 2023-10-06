package ghouls

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

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
	username     string
	password     string
)

func Serve() {
	// get HTTP Basic Auth creds from env
	getCreds()

	// get data file path
	getDataFilePath()

	// Load URLs data file
	loadDataFromFile(dataFilePath)

	// Serve static files from the "static" directory
	setupStaticAssets()

	// Handle various routes
	http.HandleFunc("/", loginHandler(rootHandler))
	http.HandleFunc("/add", loginHandler(addURL))
	http.HandleFunc("/delete", loginHandler(deleteURLs))
	http.HandleFunc("/list", loginHandler(listURLs))
	http.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("Ghouls is initialized and now listening & serving on port 8080")

	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error listening & serving on port 8080 with 10sec timeout", err)
		return
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Load the HTML template
	tmpl := template.Must(template.ParseFS(&templates.Templates, "*.html"))

	// You can pass data to the template if needed
	data := struct {
		URLs []string
	}{
		URLs: data.URLs, // Pass your list of URLs here
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

func getCreds() {
	if _, err := os.Stat(".env"); err == nil {
		// Initialize Viper from .env file
		viper.SetConfigFile(".env") // Specify the name of your .env file

		// Read the .env file
		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("Error reading .env file: %s\n", err)
			os.Exit(1)
		}
	}

	// Enable reading environment variables
	viper.AutomaticEnv()

	// get HTTP Basic Auth username and password from Viper
	username = viper.GetString("BASIC_AUTH_USERNAME")
	password = viper.GetString("BASIC_AUTH_PASSWORD")
	if username == "" {
		fmt.Println("basic auth username must be provided")
		os.Exit(1)
	}

	if password == "" {
		fmt.Println("basic auth password must be provided")
		os.Exit(1)
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

		// Check if the URL starts with "http://" or "https://"
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			// If it doesn't start with either, prepend "https://"
			url = "https://" + url
		}

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
