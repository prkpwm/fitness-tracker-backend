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
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

var fitnessRecords []FitnessData
const dataDir = "fitness_data"
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
	// Load .env file
	godotenv.Load()
	
	loadData()
	

	
	r := mux.NewRouter()
	
	r.HandleFunc("/api/fitness", getFitnessData).Methods("GET")
	r.HandleFunc("/api/fitness/all", getAllFitnessData).Methods("GET")
	r.HandleFunc("/api/fitness/year/{year}", getFitnessDataByYear).Methods("GET")
	r.HandleFunc("/api/fitness/year/{year}/month/{month}", getFitnessDataByMonth).Methods("GET")
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

func getFitnessDataByYear(w http.ResponseWriter, r *http.Request) {
	if fitnessRecords == nil || len(fitnessRecords) == 0 {
		loadData()
	}
	vars := mux.Vars(r)
	year := vars["year"]
	
	var yearRecords []FitnessData
	for _, record := range fitnessRecords {
		if strings.HasPrefix(record.Date, year) {
			yearRecords = append(yearRecords, record)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(yearRecords)
}

func getFitnessDataByMonth(w http.ResponseWriter, r *http.Request) {
	if fitnessRecords == nil || len(fitnessRecords) == 0 {
		loadData()
	}
	vars := mux.Vars(r)
	year := vars["year"]
	month := vars["month"]
	
	var monthRecords []FitnessData
	for _, record := range fitnessRecords {
		t, err := time.Parse("2006-01-02", record.Date)
		if err != nil {
			continue
		}
		if fmt.Sprintf("%d", t.Year()) == year && fmt.Sprintf("%02d", t.Month()) == month {
			monthRecords = append(monthRecords, record)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monthRecords)
}

func loadData() {
	fitnessRecords = []FitnessData{}
	
	// Ensure data directory exists
	os.MkdirAll(dataDir, 0755)
	
	// Load all monthly files
	years, err := os.ReadDir(dataDir)
	if err != nil {
		log.Printf("Created data directory: %s", dataDir)
		return
	}
	
	for _, year := range years {
		if !year.IsDir() {
			continue
		}
		
		yearPath := fmt.Sprintf("%s/%s", dataDir, year.Name())
		months, err := os.ReadDir(yearPath)
		if err != nil {
			continue
		}
		
		for _, month := range months {
			if !strings.HasSuffix(month.Name(), ".json") {
				continue
			}
			
			filePath := fmt.Sprintf("%s/%s", yearPath, month.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			
			var monthRecords []FitnessData
			err = json.Unmarshal(data, &monthRecords)
			if err != nil {
				continue
			}
			
			fitnessRecords = append(fitnessRecords, monthRecords...)
		}
	}
	
	log.Printf("Loaded %d records from %s", len(fitnessRecords), dataDir)
}

func saveData() {
	// Group records by year/month
	grouped := make(map[string][]FitnessData)
	
	for _, record := range fitnessRecords {
		t, err := time.Parse("2006-01-02", record.Date)
		if err != nil {
			continue
		}
		
		key := fmt.Sprintf("%d/%02d", t.Year(), t.Month())
		grouped[key] = append(grouped[key], record)
	}
	
	// Save each month's data
	for key, records := range grouped {
		parts := strings.Split(key, "/")
		year, month := parts[0], parts[1]
		
		// Ensure directory exists
		yearDir := fmt.Sprintf("%s/%s", dataDir, year)
		os.MkdirAll(yearDir, 0755)
		
		// Save month file
		filePath := fmt.Sprintf("%s/%s.json", yearDir, month)
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			continue
		}
		
		err = os.WriteFile(filePath, data, 0644)
		if err != nil {
			log.Printf("Error saving %s: %v", filePath, err)
			continue
		}
		
		// Update GitHub with monthly file
		githubPath := fmt.Sprintf("fitness_data/%s/%s.json", year, month)
		updateGitHubFile(githubPath, data)
	}
	
}

func updateGitHubFile(githubPath string, data []byte) {
	token := os.Getenv("UP_TOK")
	log.Printf("UP_TOK %s", token)
	if token == "" {
		return
	}
	
	sha := getGitHubFileSHA(token, githubPath)
	
	payload := map[string]interface{}{
		"message": fmt.Sprintf("Update %s", githubPath),
		"content": base64.StdEncoding.EncodeToString(data),
	}
	
	if sha != "" {
		payload["sha"] = sha
	}
	
	jsonPayload, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/%s", githubPath)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	
	log.Printf("GitHub API Request: %s %s", req.Method, req.URL)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GitHub API Error: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		log.Printf("Updated GitHub: %s", githubPath)
	} else {
		log.Printf("GitHub API failed: %d", resp.StatusCode)
	}
}

func getGitHubFileSHA(token, filePath string) string {
	url := fmt.Sprintf("https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/%s", filePath)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if sha, ok := result["sha"].(string); ok {
		return sha
	}
	return ""
}

