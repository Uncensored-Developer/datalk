package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func main() {
	// Get the module directory by running "go list -m -f {{.Dir}}"
	modDir, err := getModuleDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting module directory: %v\n", err)
		os.Exit(1)
	}
	modFile := filepath.Join(modDir, "etc", "tools", "go.mod")

	tools, err := parseToolsFromGoMod(modFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing tools from go.mod: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <tool>\n", os.Args[0])
		os.Exit(1)
	}

	toolName := os.Args[1]
	toolPath, ok := tools[toolName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Tool %s not found in go.mod\n", toolName)
		os.Exit(1)
	}

	// Execute the command
	cmd := exec.Command("go", append([]string{"run", "-modfile", modFile, "-mod=readonly", toolPath}, os.Args[2:]...)...) // nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

// getModuleDir runs "go list -m -f {{.Dir}}" and returns the module directory
func getModuleDir() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get module directory: %w", err)
	}

	// Trim whitespace from the output
	modDir := strings.TrimSpace(string(output))
	if modDir == "" {
		return "", fmt.Errorf("module directory is empty")
	}

	return modDir, nil
}

// parseToolsFromGoMod reads the go.mod file and extracts tools from the tool block
func parseToolsFromGoMod(modFile string) (map[string]string, error) {
	data, err := os.ReadFile(modFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod file: %w", err)
	}

	file, err := modfile.Parse(modFile, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod file: %w", err)
	}

	tools := make(map[string]string)
	for _, tool := range file.Tool {
		toolName := exeFromImportPath(tool.Path)
		if toolName != "" {
			tools[toolName] = tool.Path
		}
	}

	return tools, nil
}

// extractToolName extracts the executable name from a module path. Copied from the go source code.
func exeFromImportPath(packagePath string) string {
	_, elem := path.Split(packagePath)
	// If this is example.com/mycmd/v2, it's more useful to
	// install it as mycmd than as v2. See golang.org/issue/24667.
	if elem != packagePath && isVersionElement(elem) {
		_, elem = path.Split(path.Dir(packagePath))
	}
	return elem
}

// isVersionElement reports whether s is a well-formed path version element:
// v2, v3, v10, etc, but not v0, v05, v1. Copied from the go source code.
func isVersionElement(s string) bool {
	if len(s) < 2 || s[0] != 'v' || s[1] == '0' || s[1] == '1' && len(s) == 2 {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || '9' < s[i] {
			return false
		}
	}
	return true
}
