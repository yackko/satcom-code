// types/satellite.go
package types

// Satellite represents information about an Earth satellite.
type Satellite struct {
	Name             string  `json:"name"`
	OrbitType        string  `json:"orbitType"`
	Altitude         float64 `json:"altitude"`
	Eccentricity     float64 `json:"eccentricity"`
	Inclination      float64 `json:"inclination"`
	PowerSystem      string  `json:"powerSystem"`
	Communication    string  `json:"communication"`
	Size             float64 `json:"size"`
	Weight           float64 `json:"weight"`
	Constellation    bool    `json:"constellation"`
	RemoteSensing    string  `json:"remoteSensing"`
	LaunchDate       string  `json:"launchDate"` // Format: YYYY-MM-DD
	Operator         string  `json:"operator"`
	MissionObjective string  `json:"missionObjective"`
	Status           string  `json:"status"` // e.g., Active, Inactive
}
