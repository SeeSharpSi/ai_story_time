package templates

import "strings"

// HealthStatus represents the player's health state and its corresponding color.
type HealthStatus struct {
	Description string
	Color       string
}

// GetHealthStatus returns a HealthStatus struct based on the player's health percentage.
func GetHealthStatus(health int) HealthStatus {
	switch {
	case health >= 80:
		return HealthStatus{"Healthy", "#a6e22e"} // Lime Green
	case health >= 50:
		return HealthStatus{"Injured", "#e6db74"} // Yellow
	case health >= 20:
		return HealthStatus{"Wounded", "#fd971f"} // Orange
	case health > 0:
		return HealthStatus{"Critical", "#f92672"} // Pink/Red
	default:
		return HealthStatus{"Deceased", "#75715e"} // Gray
	}
}

// FormatProperties creates a string from a slice of item properties.
func FormatProperties(props []string) string {
	if len(props) == 0 {
		return ""
	}
	return strings.Join(props, ", ")
}
