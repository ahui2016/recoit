package ibm

import (
	"encoding/json"

	"github.com/ahui2016/recoit/cloud"
)

// Settings for IBM COS.
type Settings struct {
	Provider          cloud.Provider
	ApiKey            string
	ServiceInstanceID string // resource_instance_id
	ServiceEndpoint   string
	BucketLocation    string
	BucketName        string
}

// NewSettingsFromJSON .
func NewSettingsFromJSON(settingsJSON []byte) *Settings {
	settings := new(Settings)
	if err := json.Unmarshal(settingsJSON, settings); err != nil {
		panic(err)
	}
	return settings
}

// GetProvider .
func (settings *Settings) GetProvider() cloud.Provider {
	return settings.Provider
}

// Encode to JSON.
func (settings *Settings) Encode() []byte {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		panic(err)
	}
	return settingsJSON
}

// NewCOS .
func (settings *Settings) NewCOS() cloud.ObjectStorage {
	return &COS{
		apiKey:            settings.ApiKey,
		serviceInstanceID: settings.ServiceInstanceID,
		authEndpoint:      authEndpoint, // const
		serviceEndpoint:   settings.ServiceEndpoint,
		bucketLocation:    settings.BucketLocation,
		bucketName:        settings.BucketName,
	}
}
