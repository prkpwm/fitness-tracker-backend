package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var fitnessRecords []FitnessData

func main() {
	r := mux.NewRouter()
	
	r.HandleFunc("/api/fitness", getFitnessData).Methods("GET")
	r.HandleFunc("/api/fitness", createFitnessData).Methods("POST")
	r.HandleFunc("/api/fitness/{date}", getFitnessDataByDate).Methods("GET")
	r.HandleFunc("/get", getRawJsonByDate).Methods("GET")
	
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(r)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getFitnessData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fitnessRecords)
}

func createFitnessData(w http.ResponseWriter, r *http.Request) {
	var data FitnessData
	json.NewDecoder(r.Body).Decode(&data)
	
	if data.Date == "" {
		data.Date = time.Now().Format("2006-01-02")
	}
	
	fitnessRecords = append(fitnessRecords, data)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func getFitnessDataByDate(w http.ResponseWriter, r *http.Request) {
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