package prompts

import (
	"os"
)

func Load(commandsFile string) (string, error) {
	if commandsFile == "" {
		return System, nil
	}
	commands, err := os.ReadFile(commandsFile)
	if err != nil {
		return "", err
	}
	return System + "\n\n# Available Commands\n\n" + string(commands), nil
}

func MustLoad(commandsFile string) string {
	prompt, err := Load(commandsFile)
	if err != nil {
		panic("failed to load commands file: " + err.Error())
	}
	return prompt
}
