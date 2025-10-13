package hosts

import (
	"net/http"
	"testing"
)

func TestGetNextHost(t *testing.T) {
	// Save original functions and restore after test
	originalGetHostname := getHostnameFunc
	originalCheckHostHealth := checkHostHealthFunc
	defer func() {
		getHostnameFunc = originalGetHostname
		checkHostHealthFunc = originalCheckHostHealth
	}()

	tests := []struct {
		name             string
		hostname         string
		healthStatus     int
		expectedNextHost string
	}{
		{
			name:             "tele0 with StatusOK",
			hostname:         "tele0",
			healthStatus:     http.StatusOK,
			expectedNextHost: "tele1", // Next healthy host after tele0
		},
		{
			name:             "tele0 with StatusBadRequest",
			hostname:         "tele0",
			healthStatus:     http.StatusBadRequest,
			expectedNextHost: "tele1", // Falls back to immediate next host
		},
		{
			name:             "tele4 with StatusOK",
			hostname:         "tele4",
			healthStatus:     http.StatusOK,
			expectedNextHost: "tele0", // Wraps around to tele0
		},
		{
			name:             "tele4 with StatusBadRequest",
			hostname:         "tele4",
			healthStatus:     http.StatusBadRequest,
			expectedNextHost: "tele0", // Falls back to immediate next host (wraps around)
		},
		{
			name:             "tele33 with StatusOK",
			hostname:         "tele33",
			healthStatus:     http.StatusOK,
			expectedNextHost: "tele0", // Not in list, starts from 0
		},
		{
			name:             "tele33 with StatusBadRequest",
			hostname:         "tele33",
			healthStatus:     http.StatusBadRequest,
			expectedNextHost: "tele0", // Not in list, falls back to first host
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock getHostnameFunc
			getHostnameFunc = func() (string, error) {
				return tt.hostname, nil
			}

			// Mock checkHostHealthFunc
			checkHostHealthFunc = func(host string) bool {
				return tt.healthStatus == http.StatusOK
			}

			result := GetNextHost()

			if result != tt.expectedNextHost {
				t.Errorf("GetNextHost() = %q; want %q", result, tt.expectedNextHost)
			}
		})
	}
}
