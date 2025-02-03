package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Grafana user struct
type User struct {
	Username string `json:"login"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	// Load users.json file from environment variable
	filePath := "/etc/grafana/users.json"
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading users file: %v", err)
	}

	// Parse users JSON
	var users []User
	err = json.Unmarshal(fileData, &users)
	if err != nil {
		log.Fatalf("Error parsing users JSON: %v", err)
	}

	// Set Grafana admin credentials
	//TODO: get from k8s secret
	grafanaURL := "http://localhost:3000"
	adminUser := "admin"
	adminPass := "mypassword"

	// Create users
	for _, user := range users {
		err := createUser(grafanaURL, adminUser, adminPass, user)
		if err != nil {
			log.Printf("Failed to create user %s: %v", user.Username, err)
		} else {
			log.Printf("User %s created with role %s", user.Username, user.Role)
		}
	}
}

func createUser(grafanaURL, adminUser, adminPass string, user User) error {
	// user creation Grafana API
	apiEndpoint := fmt.Sprintf("%s/api/admin/users", grafanaURL)

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("Error marshalling user JSON: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(userJSON))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to create user, status code: %d", resp.StatusCode)
	}

	return nil
}
