package ui

// SafeWidth returns a safe width for separator lines, ensuring it's never negative.
func SafeWidth(width int) int {
	if width <= 0 {
		return 40 // Default width for headless mode
	}
	return width
}
