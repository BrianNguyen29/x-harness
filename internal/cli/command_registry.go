package cli

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed commands.json
var embeddedCommandRegistry []byte

var commands = loadCommandRegistry()

func loadCommandRegistry() []CommandInfo {
	var registry []CommandInfo
	if err := json.Unmarshal(embeddedCommandRegistry, &registry); err != nil {
		panic(fmt.Sprintf("invalid embedded CLI command registry: %v", err))
	}
	seen := map[string]bool{}
	for _, command := range registry {
		if command.Name == "" {
			panic("invalid embedded CLI command registry: command name is empty")
		}
		if command.Description == "" {
			panic(fmt.Sprintf("invalid embedded CLI command registry: %s description is empty", command.Name))
		}
		if command.Maturity == "" {
			panic(fmt.Sprintf("invalid embedded CLI command registry: %s maturity is empty", command.Name))
		}
		if seen[command.Name] {
			panic(fmt.Sprintf("invalid embedded CLI command registry: duplicate command %s", command.Name))
		}
		seen[command.Name] = true
	}
	return registry
}

func commandByName(name string) (CommandInfo, bool) {
	for _, command := range commands {
		if command.Name == name {
			return command, true
		}
	}
	return CommandInfo{}, false
}

func onboardingCommands() []CommandInfo {
	result := []CommandInfo{}
	for _, command := range commands {
		if command.Onboarding {
			result = append(result, command)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i].OnboardingOrder
		right := result[j].OnboardingOrder
		if left == right {
			return result[i].Name < result[j].Name
		}
		if left == 0 {
			return false
		}
		if right == 0 {
			return true
		}
		return left < right
	})
	return result
}
