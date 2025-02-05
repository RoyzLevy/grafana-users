package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Grafana user struct
type User struct {
	Username string `json:"login"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type OrgCheckResponse struct {
	OrgID int `json:"id"` // Ensure it's mapped to "id" (not "orgId")
}

type OrgResponse struct {
	OrgID   int    `json:"orgId"` // Change type from string to int
	Message string `json:"message"`
}

// Main function
func main() {
	log.Println("Starting Grafana user provisioning...")

	// Load users.json file
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

	grafanaURL := "localhost:3000"
	adminUser := "admin"
	adminPass := "mypassword"

	var orgID int

	// Check if organization exists and create it if necessary
	orgID, err = checkOrgExists(grafanaURL, adminUser, adminPass, "Para")
	if err != nil {
		log.Printf("Failed to check org existence: %v", err)
		return
	}

	if orgID == 0 {
		orgID, err = createOrg(grafanaURL, adminUser, adminPass, "Para")
		if err == nil {
			log.Printf("Created organization with id: %d", orgID)
		} else {
			log.Printf("Failed creating org: %v", err)
			return
		}
	}

	// Create users and modify their roles
	for _, user := range users {
		userExists, _ := checkUserExists(grafanaURL, adminUser, adminPass, user.Username)
		if !userExists {
			err := createUser(grafanaURL, adminUser, adminPass, user)
			if err != nil {
				log.Printf("Failed to create user %s: %v", user.Username, err)
			} else {
				log.Printf("User %s created", user.Username)
			}

			// If user creation was successful, modify the role
			if err == nil {
				err = modifyUserRole(grafanaURL, adminUser, adminPass, user, orgID)
				if err != nil {
					log.Printf("Failed to modify user %s role to %s: %v", user.Username, user.Role, err)
				} else {
					log.Printf("User %s role modified to %s", user.Username, user.Role)
				}
			}
		} else {
			log.Printf("User %s already exists. Skipping", user.Username)
		}
	}

	// Wait a few seconds before exiting to ensure logs are visible
	log.Println("User provisioning completed successfully. Container will now exit.")
	time.Sleep(5 * time.Second) // Optional delay for log visibility

	// Exit the container cleanly
	os.Exit(0)
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
		var org OrgCheckResponse
		err := json.NewDecoder(resp.Body).Decode(&org)
		if err != nil {
			return 0, fmt.Errorf("error decoding org response: %v", err)
		}
		return org.OrgID, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil // Org does not exist
	}

	return 0, fmt.Errorf("unexpected status code while checking org: %d", resp.StatusCode)
}

// Function to create an organization if it does not exist
func createOrg(grafanaURL, adminUser, adminPass, orgName string) (int, error) {
	apiEndpoint := fmt.Sprintf("http://%s/api/orgs", grafanaURL)

	org := map[string]string{"name": orgName}
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

	var orgResponse OrgResponse
	err = json.NewDecoder(resp.Body).Decode(&orgResponse)
	if err != nil {
		return 0, fmt.Errorf("error decoding org creation response: %v", err)
	}

	// Return the parsed orgId directly
	return orgResponse.OrgID, nil
}

// Function to check if a user exists in Grafana
func checkUserExists(grafanaURL, adminUser, adminPass, username string) (bool, error) {
	apiEndpoint := fmt.Sprintf("http://%s/api/users/lookup?login=%s", grafanaURL, username)

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request to check user existence: %v", err)
	}

	req.SetBasicAuth(adminUser, adminPass)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error sending request to check user existence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil // User exists
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil // User does not exist
	}

	return false, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
}

// Function to create a user in Grafana
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

// Function to assign a user to an organization with a specific role
func modifyUserRole(grafanaURL, adminUser, adminPass string, user User, orgID int) error {
	orgEndpoint := fmt.Sprintf("http://%s:%s@%s/api/orgs/%d/users", adminUser, adminPass, grafanaURL, orgID)
	log.Printf("Assigning role via: %s", orgEndpoint)

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
