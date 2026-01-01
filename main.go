package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var fitnessRecords []FitnessData
const dataFile = "fitness_data.json"
const backupFile = "backup.txt"

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log request
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		log.Printf("REQ: %s %s - Body: %s", r.Method, r.URL.Path, string(body))
		
		// Capture response
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapper, r)
		
		// Log response
		duration := time.Since(start)
		log.Printf("RES: %d - %s %s - %v", wrapper.statusCode, r.Method, r.URL.Path, duration)
	})
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	loadData()
	

	
	r := mux.NewRouter()
	
	r.HandleFunc("/api/fitness", getFitnessData).Methods("GET")
	r.HandleFunc("/api/fitness/all", getAllFitnessData).Methods("GET")
	r.HandleFunc("/api/fitness", createFitnessData).Methods("POST")
	r.HandleFunc("/api/fitness/{date}", getFitnessDataByDate).Methods("GET")
	r.HandleFunc("/get", getRawJsonByDate).Methods("GET")
	
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(loggingMiddleware(r))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getFitnessData(w http.ResponseWriter, r *http.Request) {
	if fitnessRecords == nil || len(fitnessRecords) == 0 {
		loadData()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fitnessRecords)
}

func getAllFitnessData(w http.ResponseWriter, r *http.Request) {
	if fitnessRecords == nil || len(fitnessRecords) == 0 {
		loadData()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fitnessRecords)
}

func createFitnessData(w http.ResponseWriter, r *http.Request) {
	var data FitnessData
	json.NewDecoder(r.Body).Decode(&data)
	
	// Check for duplicate by date and replace if exists
	for i, record := range fitnessRecords {
		if record.Date == data.Date {
			fitnessRecords[i] = data
			saveData()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
	}
	
	// If no duplicate found, append new record
	fitnessRecords = append(fitnessRecords, data)
	saveData()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func getFitnessDataByDate(w http.ResponseWriter, r *http.Request) {
	if fitnessRecords == nil || len(fitnessRecords) == 0 {
		loadData()
	}
	vars := mux.Vars(r)
	date := vars["date"]
	
	for _, record := range fitnessRecords {
		if record.Date == date {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(record)
			return
		}
	}
	
	http.NotFound(w, r)
}

func getRawJsonByDate(w http.ResponseWriter, r *http.Request) {
	if fitnessRecords == nil || len(fitnessRecords) == 0 {
		loadData()
	}
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	for _, record := range fitnessRecords {
		if record.Date == date {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(record)
			return
		}
	}
	
	http.NotFound(w, r)
}

func loadData() {
	// Try local file first
	data, err := os.ReadFile(dataFile)
	if err != nil {
		log.Println("No local file found, trying backup...")
		loadFromGitHub()
		return
	}
	
	err = json.Unmarshal(data, &fitnessRecords)
	if err != nil {
		log.Printf("Error parsing local data: %v, trying backup...", err)
		loadFromGitHub()
		return
	}
	
	// If local data is empty, try backup
	if len(fitnessRecords) == 0 {
		log.Println("Local data empty, trying backup...")
		loadFromGitHub()
	}
}

func loadFromGitHub() {
	resp, err := http.Get("https://raw.githubusercontent.com/prkpwm/fitness-tracker-backend/refs/heads/main/fitness_data.json")
	if err != nil {
		log.Printf("Error fetching from GitHub: %v", err)
		return
	}
	defer resp.Body.Close()
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading GitHub response: %v", err)
		return
	}
	
	err = json.Unmarshal(data, &fitnessRecords)
	if err != nil {
		log.Printf("Error parsing GitHub data: %v", err)
		return
	}
	
	log.Printf("Loaded %d records from GitHub", len(fitnessRecords))
}

func saveData() {
	data, err := json.MarshalIndent(fitnessRecords, "", "  ")
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return
	}
	
	err = os.WriteFile(dataFile, data, 0644)
	if err != nil {
		log.Printf("Error saving data: %v", err)
	}
	
	// Create backup
	createBackup(data)
	
	// Update GitHub
	updateGitHub(data)
}

func createBackup(data []byte) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	backupContent := fmt.Sprintf("[%s] Backup created\n%s\n\n", timestamp, string(data))
	
	err := os.WriteFile(backupFile, []byte(backupContent), 0644)
	if err != nil {
		log.Printf("Error creating backup: %v", err)
	} else {
		log.Println("Backup created successfully")
	}
}

func updateGitHub(data []byte) {
	token := os.Getenv("UP_TOK")
	if token == "" {
		log.Println("No GitHub token, skipping GitHub update")
		return
	}
	
	sha, err := getFileSHA(token)
	if err != nil {
		log.Printf("Error getting file SHA: %v", err)
		return
	}
	
	payload := map[string]interface{}{
		"message": "Update fitness data",
		"content": base64.StdEncoding.EncodeToString(data),
		"sha":     sha,
	}
	
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", "https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/fitness_data.json", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error updating GitHub: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		log.Println("Successfully updated GitHub")
	} else {
		log.Printf("GitHub update failed: %d", resp.StatusCode)
	}
}

func getFileSHA(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/fitness_data.json", nil)
	req.Header.Set("Authorization", "token "+token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	sha, ok := result["sha"].(string)
	if !ok {
		return "", fmt.Errorf("SHA not found")
	}
	
	return sha, nil
}