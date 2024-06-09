package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type RegisterRequest struct {
	SystemURI string `json:"system_uri"`
	Port      int    `json:"port"`
}

type RegisterResponse struct {
	Message string `json:"message"`
}

type InquiryResponse struct {
	Success             bool `json:"success"`
	PercentageProcessed int  `json:"percentage_processed"`
}

type SignalResponse struct {
	PercentageProcessed int `json:"percentage_processed"`
}

// Cache entry struct
type cacheEntry struct {
	systemURI string
	port      int
	timestamp time.Time
}

var (
	cache  = make(map[string]cacheEntry)
	mutex  = &sync.Mutex{}
	client = &http.Client{
		Timeout: 5 * time.Second, // Set timeout for each request
	}
)

func main() {
	http.HandleFunc("/api/register", registerHandler)
	http.HandleFunc("/api/inquiry", inquiryHandler)

	// Start a goroutine to clean up expired cache entries every hour
	go cleanupCache()

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		handleRegisterPost(w, r)
	} else if r.Method == http.MethodGet {
		handleRegisterGet(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleRegisterPost(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("Received /api/register request:")
	fmt.Printf("SystemURI: %s, Port: %d\n", req.SystemURI, req.Port)

	// Add to cache
	mutex.Lock()
	cache[req.SystemURI] = cacheEntry{
		systemURI: req.SystemURI,
		port:      req.Port,
		timestamp: time.Now(),
	}
	mutex.Unlock()

	resp := RegisterResponse{
		Message: "Success",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	fmt.Println("Sent /api/register response:")
	fmt.Printf("Message: %s\n", resp.Message)
}

func handleRegisterGet(w http.ResponseWriter, _ *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	// Create a copy of the cache to avoid race conditions
	cacheCopy := make(map[string]cacheEntry)
	for key, entry := range cache {
		cacheCopy[key] = entry
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cacheCopy)

	fmt.Println("Sent /api/register GET response with current cache state")
}

func inquiryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(10 << 20) // 10 MB limit

	wifiFile, _, err := r.FormFile("wifi_data")
	if err != nil {
		http.Error(w, "Error retrieving WiFi file", http.StatusBadRequest)
		return
	}
	defer wifiFile.Close()

	bleFile, _, err := r.FormFile("ble_data")
	if err != nil {
		http.Error(w, "Error retrieving BLE file", http.StatusBadRequest)
		return
	}
	defer bleFile.Close()

	wifiData, err := parseCSV(wifiFile)
	if err != nil {
		http.Error(w, "Error parsing WiFi CSV", http.StatusBadRequest)
		return
	}

	bleData, err := parseCSV(bleFile)
	if err != nil {
		http.Error(w, "Error parsing BLE CSV", http.StatusBadRequest)
		return
	}

	fmt.Println("Received /api/inquiry request:")
	fmt.Println("WiFi Data:")
	for _, row := range wifiData {
		fmt.Println(row)
	}

	fmt.Println("BLE Data:")
	for _, row := range bleData {
		fmt.Println(row)
	}

	// Send POST request to each system_uri in the cache
	maxPercentage := 0
	var wg sync.WaitGroup
	responseChan := make(chan int, len(cache))

	mutex.Lock()
	for _, entry := range cache {
		wg.Add(1)
		go func(systemURI string, port int) {
			defer wg.Done()
			percentage := querySystem(systemURI, port)
			responseChan <- percentage
		}(entry.systemURI, entry.port)
	}
	mutex.Unlock()

	wg.Wait()
	close(responseChan)

	for percentage := range responseChan {
		if percentage > maxPercentage {
			maxPercentage = percentage
		}
	}

	resp := InquiryResponse{
		Success:             true,
		PercentageProcessed: maxPercentage,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	fmt.Println("Sent /api/inquiry response:")
	fmt.Printf("Success: %t, PercentageProcessed: %d\n", resp.Success, resp.PercentageProcessed)
}

func querySystem(systemURI string, port int) int {
	url := fmt.Sprintf("%s:%d/api/signals/server", systemURI, port)
	signalRequest := map[string]string{
		"data": "dummy_data", // Replace with actual data if needed
	}
	requestBody, err := json.Marshal(signalRequest)
	if err != nil {
		fmt.Printf("Error marshaling request for %s: %v\n", systemURI, err)
		return 0
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", systemURI, err)
		return 0
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request to %s: %v\n", systemURI, err)
		return 0
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response from %s: %v\n", systemURI, err)
		return 0
	}

	var signalResponse SignalResponse
	if err := json.Unmarshal(body, &signalResponse); err != nil {
		fmt.Printf("Error unmarshaling response from %s: %v\n", systemURI, err)
		return 0
	}

	return signalResponse.PercentageProcessed
}

func parseCSV(file io.Reader) ([][]string, error) {
	reader := csv.NewReader(file)
	var data [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data = append(data, record)
	}
	return data, nil
}

func cleanupCache() {
	for {
		time.Sleep(1 * time.Hour)
		mutex.Lock()
		now := time.Now()
		for key, entry := range cache {
			if now.Sub(entry.timestamp) > 24*time.Hour {
				delete(cache, key)
				fmt.Printf("Deleted expired cache entry: %s\n", key)
			}
		}
		mutex.Unlock()
	}
}
