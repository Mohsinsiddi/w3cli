package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user with a yes/no question. Returns true for yes.
func Confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", StyleWarning.Render(prompt))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

// ConfirmDanger is like Confirm but styled with the error color (for destructive actions).
func ConfirmDanger(prompt string) bool {
	fmt.Printf("%s [y/N]: ", StyleError.Render("⚠ "+prompt))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
