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

type OrgResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Main function
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
	grafanaURL := "localhost:3000" // Updated to use the full service name
	adminUser := "admin"
	adminPass := "mypassword"

	var orgID int

	// Check if organization exists and if not create it
	orgID, err = checkOrgExists(grafanaURL, adminUser, adminPass, "Para")
	if err != nil {
		log.Printf("Failed to check org existence: %v", err)
		return
	}

	if orgID == 0 {
		orgID, err = createOrg(grafanaURL, adminUser, adminPass, "Para")
		if err != nil {
			log.Printf("Created organization with id: %d", orgID)
		} else {
			log.Printf("Failed creating org: %v", err)
			return
		}
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

// Function to check if the organization exists and return orgID
func checkOrgExists(grafanaURL, adminUser, adminPass, orgName string) (int, error) {
	apiEndpoint := fmt.Sprintf("http://%s/api/orgs/name/%s", grafanaURL, orgName)

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.SetBasicAuth(adminUser, adminPass)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending HTTP request to check org existence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Parse the organization response
		var org OrgResponse
		err := json.NewDecoder(resp.Body).Decode(&org)
		if err != nil {
			return 0, fmt.Errorf("error decoding org response: %v", err)
		}
		return org.ID, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Org does not exist
		return 0, nil
	}

	return 0, fmt.Errorf("unexpected status code while checking org: %d", resp.StatusCode)
}

// Function to create the organization if it does not exist
func createOrg(grafanaURL, adminUser, adminPass, orgName string) (int, error) {
	apiEndpoint := fmt.Sprintf("http://%s/api/orgs", grafanaURL)

	org := map[string]string{
		"name": orgName,
	}

	orgJSON, err := json.Marshal(org)
	if err != nil {
		return 0, fmt.Errorf("error marshalling organization JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(orgJSON))
	if err != nil {
		return 0, fmt.Errorf("error creating HTTP request to create org: %v", err)
	}

	req.SetBasicAuth(adminUser, adminPass)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending HTTP request to create org: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to create organization, status code: %d", resp.StatusCode)
	}

	// Parse the created org response to get org ID
	var orgResponse OrgResponse
	err = json.NewDecoder(resp.Body).Decode(&orgResponse)
	if err != nil {
		return 0, fmt.Errorf("error decoding org creation response: %v", err)
	}

	return orgResponse.ID, nil
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
	log.Printf("orgEndpoint: %s", orgEndpoint)

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
