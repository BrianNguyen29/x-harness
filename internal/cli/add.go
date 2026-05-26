package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func handleAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	module := ""
	values := ""
	outPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: add <module> [values] --out <path>")
				return ExitUsage
			}
			outPath = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				WriteLine(stderr, "unknown flag: %s", args[i])
				return ExitUsage
			}
			if module == "" {
				module = args[i]
			} else if values == "" {
				values = args[i]
			} else {
				WriteLine(stderr, "usage: add <module> [values] --out <path>")
				return ExitUsage
			}
		}
	}

	validModules := map[string]bool{
		"claim":           true,
		"evidence":        true,
		"completion-card": true,
	}

	if module == "" || !validModules[module] {
		WriteLine(stderr, "usage: add <module> [values] --out <path>")
		return ExitUsage
	}

	if outPath == "" {
		outPath = module + ".yaml"
	}

	data := make(map[string]interface{})
	data["id"] = strings.ToUpper(module) + "-" + fmt.Sprintf("%d", time.Now().UnixMilli())
	data["created_at"] = time.Now().UTC().Format(time.RFC3339)

	if values != "" {
		for _, pair := range strings.Split(values, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 && parts[0] != "" {
				data[parts[0]] = parts[1]
			}
		}
	}

	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		WriteLine(stderr, "failed to marshal YAML: %v", err)
		return ExitError
	}

	if err := os.WriteFile(outPath, yamlBytes, 0644); err != nil {
		WriteLine(stderr, "failed to write file: %v", err)
		return ExitError
	}

	WriteLine(stdout, "Added %s -> %s", module, outPath)
	return ExitOK
}
