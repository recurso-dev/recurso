package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	baseURL = "http://localhost:8080/v1"
	apiKey  = "sk_test_12345"
)

func main() {
	log.Println("🚀 Starting Demo Activity Generator...")

	// 1. Create a Customer
	customerName := fmt.Sprintf("Demo User %d", time.Now().Unix()%1000)
	customerID := createCustomer(customerName)
	log.Printf("✅ Created Customer: %s (%s)", customerName, customerID)

	// 2. Fetch Plans to pick one
	planID := getFirstPlan()
	if planID == "" {
		log.Fatal("❌ No plans found! Seed data first.")
	}
	log.Printf("ℹ️ Using Plan ID: %s", planID)

	// 3. Create Subscription
	subID := createSubscription(customerID, planID)
	log.Printf("✅ Created Subscription: %s", subID)

	log.Println("🎉 Data created! Check your Dashboard Activity Feed.")
}

func createCustomer(name string) string {
	payload := map[string]interface{}{
		"name":  name,
		"email": fmt.Sprintf("demo.%d@example.com", time.Now().UnixNano()),
		"address": map[string]string{
			"line1":   "123 Startup Way",
			"city":    "San Francisco",
			"state":   "CA",
			"country": "US",
		},
	}
	return post("/customers", payload)["id"].(string)
}

func getFirstPlan() string {
	resp := get("/plans")
	data := resp["data"].([]interface{})
	if len(data) == 0 {
		return ""
	}
	return data[0].(map[string]interface{})["id"].(string)
}

func createSubscription(custID, planID string) string {
	payload := map[string]string{
		"customer_id": custID,
		"plan_id":     planID,
	}
	return post("/subscriptions", payload)["id"].(string)
}

// Helpers

func post(endpoint string, body interface{}) map[string]interface{} {
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+endpoint, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("API Error (%d): %s", resp.StatusCode, string(b))
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res
}

func get(endpoint string) map[string]interface{} {
	req, _ := http.NewRequest("GET", baseURL+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res
}
