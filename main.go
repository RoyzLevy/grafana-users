package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	fileData, err := os.ReadFile(filePath)
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
	grafanaURL := "grafana:80"
	adminUser := "admin"
	adminPass := "mypassword"

	var orgID int

	// Check if organization exists and if not create it
	orgExists, err := checkOrgExists(grafanaURL, adminUser, adminPass, "Para")
	if err != nil {
		log.Printf("Failed to do org check: %v", err)
		return
	}

	if !orgExists {
		orgID, _ = createOrg(grafanaURL, adminUser, adminPass, "Para")
	}

	// Create users
	for _, user := range users {
		err := createUser(grafanaURL, adminUser, adminPass, user)
		if err != nil {
			log.Printf("Failed to create user %s: %v", user.Username, err)
		} else {
			log.Printf("User %s created", user.Username)
		}
		// if the user was created - modify org role
		if err == nil {
			err = modifyUserRole(grafanaURL, adminUser, adminPass, user, orgID)
			if err != nil {
				log.Printf("Failed to modify user %s role to %s: %v", user.Username, user.Role, err)
			} else {
				log.Printf("User %s role modified to %s", user.Username, user.Role)
			}
		}
	}
}

// Function to check if the organization exists
func checkOrgExists(grafanaURL, adminUser, adminPass, orgName string) (bool, error) {
	apiEndpoint := fmt.Sprintf("http://%s/api/orgs/name/%s", grafanaURL, orgName)

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return false, fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error sending HTTP request to check org existence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Org exists
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Org does not exist
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code while checking org: %d", resp.StatusCode)
}

// Function to create the organization if it does not exist
func createOrg(grafanaURL, adminUser, adminPass, orgName string) (int, error) {
	apiEndpoint := fmt.Sprintf("http://%s/api/orgs", grafanaURL)

	org := map[string]string{
		"name": orgName,
	}

	orgJSON, err := json.Marshal(org)
	if err != nil {
		return 1, fmt.Errorf("error marshalling organization JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(orgJSON))
	if err != nil {
		return 1, fmt.Errorf("error creating HTTP request to create org: %v", err)
	}

	req.SetBasicAuth(adminUser, adminPass)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 1, fmt.Errorf("error sending HTTP request to create org: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1, fmt.Errorf("failed to create organization, status code: %d", resp.StatusCode)
	}

	return 2, nil
}

// Step 1: Create User via Grafana API
func createUser(grafanaURL, adminUser, adminPass string, user User) error {
	apiEndpoint := fmt.Sprintf("http://%s/api/admin/users", grafanaURL)

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error marshalling user JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(userJSON))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request to create user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	return nil
}

// Step 2: Add the user to the organization with the specified role
func modifyUserRole(grafanaURL, adminUser, adminPass string, user User, orgID int) error {
	// Embed the admin credentials directly in the URL
	orgEndpoint := fmt.Sprintf("http://%s:%s@%s/api/orgs/%d/users", adminUser, adminPass, grafanaURL, orgID)

	// Prepare role assignment JSON
	roleAssignment := map[string]interface{}{
		"loginOrEmail": user.Email,
		"role":         user.Role,
	}

	roleJSON, err := json.Marshal(roleAssignment)
	if err != nil {
		return fmt.Errorf("error marshalling role assignment JSON: %v", err)
	}

	req, err := http.NewRequest("POST", orgEndpoint, bytes.NewBuffer(roleJSON))
	if err != nil {
		return fmt.Errorf("error creating HTTP request for role assignment: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request to assign role: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to assign role to user, status code: %d", resp.StatusCode)
	}

	return nil
}
