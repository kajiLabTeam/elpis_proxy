package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type RegisterRequest struct {
	SystemURI string `json:"system_uri"`
}

type RegisterResponse struct {
	Message string `json:"message"`
}

type InquiryResponse struct {
	Success             bool `json:"success"`
	PercentageProcessed int  `json:"percentage_processed"`
}

func main() {
	http.HandleFunc("/api/register", registerHandler)
	http.HandleFunc("/api/inquiry", inquiryHandler)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("Received /api/register request:")
	fmt.Printf("SystemURI: %s\n", req.SystemURI)

	resp := RegisterResponse{
		Message: "Success",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	fmt.Println("Sent /api/register response:")
	fmt.Printf("Message: %s\n", resp.Message)
}

func inquiryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(10 << 20) // 10 MB の制限

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

	resp := InquiryResponse{
		Success:             true,
		PercentageProcessed: 100,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	fmt.Println("Sent /api/inquiry response:")
	fmt.Printf("Success: %t, PercentageProcessed: %d\n", resp.Success, resp.PercentageProcessed)
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
