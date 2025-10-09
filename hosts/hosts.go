package hosts

import (
	"log"
	"net/http"
	"os"
)

var hostsList = []string{"tele0", "tele1", "tele2", "tele3", "tele4"}
var hostMsgMap = map[string]string{
	"tele0": "http://tele0:8080/message",
	"tele1": "http://tele1:8081/message",
	"tele2": "http://tele2:8082/message",
	"tele3": "http://tele3:8083/message",
	"tele4": "http://tele4:8084/message",
}
var hostHealthMap = map[string]string{
	"tele0": "http://tele0:8080/health",
	"tele1": "http://tele1:8081/health",
	"tele2": "http://tele2:8082/health",
	"tele3": "http://tele3:8083/health",
	"tele4": "http://tele4:8084/health",
}

// GetNextHost returns the next healthy host in the rotation
func GetNextHost() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Error getting hostname: %v", err)
		return ""
	}

	currentIndex := -1
	for i, host := range hostsList {
		if host == hostname {
			currentIndex = i
			break
		}
	}

	// If current hostname not found in array, start from 0
	if currentIndex == -1 {
		log.Printf("Hostname not found in array, starting from 0")
		currentIndex = -1

	}

	// Try each host starting from the next one
	for i := 1; i <= len(hostsList); i++ {
		nextIndex := (currentIndex + i) % len(hostsList)
		nextHost := hostsList[nextIndex]

		// Check health of this host
		if CheckHostHealth(nextHost) {
			return nextHost
		}
	}

	// If no healthy host found, return the immediate next host anyway
	nextIndex := (currentIndex + 1) % len(hostsList)
	return hostsList[nextIndex]
}

// CheckHostHealth checks if a given host is healthy
func CheckHostHealth(host string) bool {
	healthURL, exists := hostHealthMap[host]
	if !exists {
		return false
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetNextHostURL returns the message URL for the next host in the rotation
func GetNextHostURL() string {
	nextHost := GetNextHost()

	// If next host is the first in the list, we've completed the cycle
	if nextHost == hostsList[0] {
		return ""
	}

	if url, exists := hostMsgMap[nextHost]; exists {
		return url
	}
	return ""
}

// GetNextHostHealth checks the health of the next host in the rotation
func GetNextHostHealth() bool {
	nextHost := GetNextHost()
	healthURL, exists := hostHealthMap[nextHost]
	if !exists {
		log.Printf("Health check failed for %s: host not found in health map", nextHost)
		return false
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		log.Printf("Health check failed for %s: %v", nextHost, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check failed for %s: status %d", nextHost, resp.StatusCode)
		return false
	}

	return true
}
