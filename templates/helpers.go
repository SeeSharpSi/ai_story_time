package templates

import (
	"fmt"
	"strings"
)

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

// VignetteStyle generates a CSS style for the vignette effect based on world tension.
func VignetteStyle(tension int) string {
	opacity := float64(tension) / 200.0 // Scale opacity from 0.0 to 0.5
	spread := tension / 2
	blur := tension / 4

	style := fmt.Sprintf(`
		<style>
			#story-container::before {
				content: '';
				position: absolute;
				top: 0;
				left: 0;
				right: 0;
				bottom: 0;
				box-shadow: inset 0 0 %dpx %dpx rgba(0,0,0,%.2f);
				transition: box-shadow 0.5s ease-in-out;
				pointer-events: none;
				border-radius: 8px;
			}
		</style>
	`, blur, spread, opacity)

	return style
}