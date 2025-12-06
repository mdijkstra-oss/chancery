package prompts

import (
	"os"

	"hermes-logos/internal/lib/utils"
)

type LoadResult struct {
	Combined      string
	SystemTokens  int
	CommandTokens int
}

func Load(commandsFile string) (LoadResult, error) {
	systemTokens := utils.EstimateTokens(System)

	if commandsFile == "" {
		return LoadResult{
			Combined:      System,
			SystemTokens:  systemTokens,
			CommandTokens: 0,
		}, nil
	}

	commands, err := os.ReadFile(commandsFile)
	if err != nil {
		return LoadResult{}, err
	}

	commandTokens := utils.EstimateTokens(string(commands))
	combined := System + "\n\n# Available Commands\n\n" + string(commands)

	return LoadResult{
		Combined:      combined,
		SystemTokens:  systemTokens,
		CommandTokens: commandTokens,
	}, nil
}

func MustLoad(commandsFile string) LoadResult {
	result, err := Load(commandsFile)
	if err != nil {
		panic("failed to load commands file: " + err.Error())
	}
	return result
}
