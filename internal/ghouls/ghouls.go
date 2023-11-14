package ghouls

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/toozej/ghouls/assets"
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
	// set defaults
	localDev = false

	// get config items from env
	getEnvVars()

	// get data file path
	getDataFilePath()

	// Load URLs data file
	loadDataFromFile(dataFilePath)

	// setup router
	r, _ := setupRouter()

	// Serve static files from the "static" directory
	setupStaticAssets(r)

	// Handle various routes
	r.Get("/", loginHandler(rootHandler))
	r.Post("/add", loginHandler(addURL))
	r.Post("/delete", loginHandler(deleteURLs))
	r.Post("/list", loginHandler(listURLs))
	r.Get("/health", healthHandler)

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      r,
	}

	fmt.Println("Ghouls is initialized and now listening & serving on port 8080")

	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error listening & serving on port 8080 with 10sec timeout", err)
		return
	}
}

func setupStaticAssets(router *chi.Mux) {
	// serve regular static assets
	fs := &assets.Assets
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(fs))))
}
