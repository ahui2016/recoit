package ibm

import (
	"encoding/base64"
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

// Encode to JSON, and encode to base64.
func (settings *Settings) Encode() string {
	settingsJson, err := json.Marshal(settings)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(settingsJson)
}

func NewSettingsFromJSON64(settings64 string) *Settings {
	settingsJSON, err := base64.StdEncoding.DecodeString(settings64)
	if err != nil {
		panic(err)
	}
	settings := new(Settings)
	if err := json.Unmarshal(settingsJSON, settings); err != nil {
		panic(err)
	}
	return settings
}
