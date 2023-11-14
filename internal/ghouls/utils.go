package ghouls

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

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

func getEnvVars() {
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

	// get CSRF key and local development true/false from Viper
	csrfKey = []byte(viper.GetString("CSRF_SECRET_KEY"))
	if len(csrfKey) == 0 {
		fmt.Println("CSRF secret key must be provided")
		os.Exit(1)
	}
	localDev = viper.GetBool("LOCAL_DEV")
}

func isValidURL(inputURL string) bool {
	// Check if the URL starts with "http://" or "https://"
	if !strings.HasPrefix(inputURL, "http://") && !strings.HasPrefix(inputURL, "https://") {
		// If it doesn't start with either, prepend "https://"
		inputURL = "https://" + inputURL
	}

	// Ensure the URL is valid
	_, err := url.ParseRequestURI(inputURL)
	return err == nil
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
