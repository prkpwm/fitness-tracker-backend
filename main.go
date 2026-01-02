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
	// Get today's date
	t := time.Now()
	
	// Load data from GitHub for today's daily file
	githubPath := fmt.Sprintf("fitness_data/%d/%02d/%02d.json", t.Year(), t.Month(), t.Day())
	dailyRecord := loadFromGitHubFile(githubPath)
	
	if len(dailyRecord) > 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dailyRecord[0])
		return
	}
	
	http.NotFound(w, r)
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
	
	// Set last update timestamp
	data.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	
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
	vars := mux.Vars(r)
	date := vars["date"]
	
	// Parse date to determine year/month/day
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}
	
	// Load data from GitHub for specific daily file
	githubPath := fmt.Sprintf("fitness_data/%d/%02d/%02d.json", t.Year(), t.Month(), t.Day())
	dailyRecord := loadFromGitHubFile(githubPath)
	
	if len(dailyRecord) > 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dailyRecord[0])
		return
	}
	
	http.NotFound(w, r)
}

func getRawJsonByDate(w http.ResponseWriter, r *http.Request) {
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
	vars := mux.Vars(r)
	year := vars["year"]
	
	// Load data from GitHub for specific year by traversing all months
	yearRecords := loadFromGitHubByYearPath(fmt.Sprintf("fitness_data/%s", year))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(yearRecords)
}

func getFitnessDataByMonth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year := vars["year"]
	month := vars["month"]
	
	// Load data from GitHub for specific month directory
	monthRecords := loadFromGitHubByPath(fmt.Sprintf("fitness_data/%s/%s", year, month))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monthRecords)
}

func loadData() {
	fitnessRecords = []FitnessData{}
	
	// Load data from GitHub
	loadFromGitHub()
	
	log.Printf("Loaded %d records from GitHub", len(fitnessRecords))
}

func saveData() {
	log.Printf("Starting saveData for %d records", len(fitnessRecords))
	// Update GitHub with daily files
	for _, record := range fitnessRecords {
		t, err := time.Parse("2006-01-02", record.Date)
		if err != nil {
			log.Printf("Error parsing date %s: %v", record.Date, err)
			continue
		}
		
		data, err := json.MarshalIndent([]FitnessData{record}, "", "  ")
		if err != nil {
			log.Printf("Error marshaling data for %s: %v", record.Date, err)
			continue
		}
		
		githubPath := fmt.Sprintf("fitness_data/%d/%02d/%02d.json", t.Year(), t.Month(), t.Day())
		log.Printf("Saving record for date %s to %s", record.Date, githubPath)
		ensureGitHubDirectories(githubPath)
		updateGitHubFile(githubPath, data)
	}
}

func ensureGitHubDirectories(filePath string) {
	log.Printf("Ensuring directories exist for %s", filePath)
	token := os.Getenv("UP_TOK")
	if token == "" {
		return
	}
	
	parts := strings.Split(filePath, "/")
	for i := 1; i < len(parts)-1; i++ {
		dirPath := strings.Join(parts[:i+1], "/")
		if !checkGitHubPathExists(token, dirPath) {
			createGitHubDirectory(token, dirPath)
		}
	}
}

func checkGitHubPathExists(token, path string) bool {
	log.Printf("GitHub API Request: GET %s", path)
	url := fmt.Sprintf("https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/%s", path)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func createGitHubDirectory(token, dirPath string) {
	log.Printf("GitHub API Request: PUT %s/.gitkeep", dirPath)
	readmePath := fmt.Sprintf("%s/.gitkeep", dirPath)
	payload := map[string]interface{}{
		"message": fmt.Sprintf("Create directory %s", dirPath),
		"content": base64.StdEncoding.EncodeToString([]byte("")),
	}
	
	jsonPayload, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/%s", readmePath)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to create directory %s: %v", dirPath, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 201 {
		log.Printf("Created directory: %s", dirPath)
	}
}

func updateGitHubFile(githubPath string, data []byte) {
	token := os.Getenv("UP_TOK")
	if token == "" {
		return
	}
	
	log.Printf("GitHub API Request: PUT %s", githubPath)
	
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

func loadFromGitHub() {
	start := time.Now()
	log.Printf("[%s] Starting full GitHub data load", start.Format("15:04:05"))
	token := os.Getenv("UP_TOK")
	if token == "" {
		log.Printf("[%s] No GitHub token available", time.Now().Format("15:04:05"))
		return
	}
	
	// Get fitness_data directory contents
	years := getGitHubDirectoryContents(token, "fitness_data")
	log.Printf("[%s] Found %d years in GitHub", time.Now().Format("15:04:05"), len(years))
	
	for _, year := range years {
		// Get year directory contents
		months := getGitHubDirectoryContents(token, fmt.Sprintf("fitness_data/%s", year))
		log.Printf("[%s] Found %d months in year %s", time.Now().Format("15:04:05"), len(months), year)
		
		for _, month := range months {
			// Get month directory contents (daily files)
			days := getGitHubDirectoryContents(token, fmt.Sprintf("fitness_data/%s/%s", year, month))
			log.Printf("[%s] Found %d days in %s/%s", time.Now().Format("15:04:05"), len(days), year, month)
			
			for _, day := range days {
				if !strings.HasSuffix(day, ".json") {
					continue
				}
				
				githubPath := fmt.Sprintf("fitness_data/%s/%s/%s", year, month, day)
				url := fmt.Sprintf("https://raw.githubusercontent.com/prkpwm/fitness-tracker-backend/main/%s", githubPath)
				
				resp, err := http.Get(url)
				if err != nil || resp.StatusCode != 200 {
					log.Printf("[%s] Failed to load %s (status: %d)", time.Now().Format("15:04:05"), githubPath, resp.StatusCode)
					if resp != nil {
						resp.Body.Close()
					}
					continue
				}
				
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					log.Printf("[%s] Error reading %s: %v", time.Now().Format("15:04:05"), githubPath, err)
					continue
				}
				
				var dailyRecords []FitnessData
				err = json.Unmarshal(data, &dailyRecords)
				if err != nil {
					log.Printf("[%s] Error parsing %s: %v", time.Now().Format("15:04:05"), githubPath, err)
					continue
				}
				
				log.Printf("[%s] Loaded %d records from %s", time.Now().Format("15:04:05"), len(dailyRecords), githubPath)
				fitnessRecords = append(fitnessRecords, dailyRecords...)
			}
		}
	}
	duration := time.Since(start)
	log.Printf("[%s] Completed full GitHub data load: %d total records (took %v)", time.Now().Format("15:04:05"), len(fitnessRecords), duration)
}

func getGitHubDirectoryContents(token, path string) []string {
	start := time.Now()
	log.Printf("[%s] Getting GitHub directory contents: %s", start.Format("15:04:05"), path)
	url := fmt.Sprintf("https://api.github.com/repos/prkpwm/fitness-tracker-backend/contents/%s", path)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Printf("[%s] Failed to get directory contents: %s (status: %d)", time.Now().Format("15:04:05"), path, resp.StatusCode)
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	
	var contents []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&contents)
	
	var names []string
	for _, item := range contents {
		if name, ok := item["name"].(string); ok {
			names = append(names, name)
		}
	}
	
	duration := time.Since(start)
	log.Printf("[%s] Found %d items in %s (took %v)", time.Now().Format("15:04:05"), len(names), path, duration)
	return names
}

func loadFromGitHubByPath(path string) []FitnessData {
	start := time.Now()
	log.Printf("[%s] Loading from GitHub by path: %s", start.Format("15:04:05"), path)
	token := os.Getenv("UP_TOK")
	if token == "" {
		log.Printf("[%s] No GitHub token available", time.Now().Format("15:04:05"))
		return nil
	}
	
	var records []FitnessData
	
	// Get directory contents (daily files)
	files := getGitHubDirectoryContents(token, path)
	
	for _, file := range files {
		if !strings.HasSuffix(file, ".json") {
			continue
		}
		
		githubPath := fmt.Sprintf("%s/%s", path, file)
		url := fmt.Sprintf("https://raw.githubusercontent.com/prkpwm/fitness-tracker-backend/main/%s", githubPath)
		
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != 200 {
			log.Printf("[%s] Failed to load file %s (status: %d)", time.Now().Format("15:04:05"), githubPath, resp.StatusCode)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[%s] Error reading file %s: %v", time.Now().Format("15:04:05"), githubPath, err)
			continue
		}
		
		var fileRecords []FitnessData
		err = json.Unmarshal(data, &fileRecords)
		if err != nil {
			log.Printf("[%s] Error parsing file %s: %v", time.Now().Format("15:04:05"), githubPath, err)
			continue
		}
		
		log.Printf("[%s] Loaded %d records from %s", time.Now().Format("15:04:05"), len(fileRecords), githubPath)
		records = append(records, fileRecords...)
	}
	
	duration := time.Since(start)
	log.Printf("[%s] Completed loading from path %s: %d total records (took %v)", time.Now().Format("15:04:05"), path, len(records), duration)
	return records
}

func loadFromGitHubFile(githubPath string) []FitnessData {
	start := time.Now()
	log.Printf("[%s] Loading from GitHub: %s", start.Format("15:04:05"), githubPath)
	url := fmt.Sprintf("https://raw.githubusercontent.com/prkpwm/fitness-tracker-backend/main/%s", githubPath)
	
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		log.Printf("[%s] Failed to load from GitHub: %s (status: %d)", time.Now().Format("15:04:05"), githubPath, resp.StatusCode)
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Printf("[%s] Error reading GitHub file: %s - %v", time.Now().Format("15:04:05"), githubPath, err)
		return nil
	}
	
	var records []FitnessData
	err = json.Unmarshal(data, &records)
	if err != nil {
		log.Printf("[%s] Error parsing GitHub file: %s - %v", time.Now().Format("15:04:05"), githubPath, err)
		return nil
	}
	
	duration := time.Since(start)
	log.Printf("[%s] Successfully loaded %d records from GitHub: %s (took %v)", time.Now().Format("15:04:05"), len(records), githubPath, duration)
	return records
}

func loadFromGitHubByYearPath(yearPath string) []FitnessData {
	start := time.Now()
	log.Printf("[%s] Loading from GitHub by year path: %s", start.Format("15:04:05"), yearPath)
	token := os.Getenv("UP_TOK")
	if token == "" {
		log.Printf("[%s] No GitHub token available", time.Now().Format("15:04:05"))
		return nil
	}

	var records []FitnessData

	// Get month directories
	months := getGitHubDirectoryContents(token, yearPath)
	log.Printf("[%s] Found %d months in %s", time.Now().Format("15:04:05"), len(months), yearPath)

	for _, month := range months {
		monthPath := fmt.Sprintf("%s/%s", yearPath, month)
		monthRecords := loadFromGitHubByPath(monthPath)
		records = append(records, monthRecords...)
	}

	duration := time.Since(start)
	log.Printf("[%s] Completed loading from year path %s: %d total records (took %v)", time.Now().Format("15:04:05"), yearPath, len(records), duration)
	return records
}