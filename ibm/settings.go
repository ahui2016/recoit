package ibm

import (
	"encoding/json"
)

// Settings for IBM COS.
type Settings struct {
	ApiKey            string
	ServiceInstanceID string // resource_instance_id
	ServiceEndpoint   string
	BucketLocation    string
	BucketName        string
}

// Encode to JSON.
func (settings *Settings) Encode() []byte {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		panic(err)
	}
	return settingsJSON
}

func NewSettingsFromJSON(settingsJSON []byte) *Settings {
	settings := new(Settings)
	if err := json.Unmarshal(settingsJSON, settings); err != nil {
		panic(err)
	}
	return settings
}
