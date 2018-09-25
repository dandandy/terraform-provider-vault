package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
)

var _ cli.Command = (*PluginInfoCommand)(nil)
var _ cli.CommandAutocomplete = (*PluginInfoCommand)(nil)

type PluginInfoCommand struct {
	*BaseCommand
}

func (c *PluginInfoCommand) Synopsis() string {
	return "Read information about a plugin in the catalog"
}

func (c *PluginInfoCommand) Help() string {
	helpText := `
Usage: vault plugin info [options] NAME

  Displays information about a plugin in the catalog with the given name. If
  the plugin does not exist, an error is returned.

  Get info about a plugin:

      $ vault plugin info mysql-database-plugin

` + c.Flags().Help()

	return strings.TrimSpace(helpText)
}

func (c *PluginInfoCommand) Flags() *FlagSets {
	return c.flagSet(FlagSetHTTP | FlagSetOutputField | FlagSetOutputFormat)
}

func (c *PluginInfoCommand) AutocompleteArgs() complete.Predictor {
	return c.PredictVaultPlugins()
}

func (c *PluginInfoCommand) AutocompleteFlags() complete.Flags {
	return c.Flags().Completions()
}

func (c *PluginInfoCommand) Run(args []string) int {
	f := c.Flags()

	if err := f.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	args = f.Args()
	switch {
	case len(args) < 1:
		c.UI.Error(fmt.Sprintf("Not enough arguments (expected 1, got %d)", len(args)))
		return 1
	case len(args) > 1:
		c.UI.Error(fmt.Sprintf("Too many arguments (expected 1, got %d)", len(args)))
		return 1
	}

	client, err := c.Client()
	if err != nil {
		c.UI.Error(err.Error())
		return 2
	}

	pluginName := strings.TrimSpace(args[0])

	resp, err := client.Sys().GetPlugin(&api.GetPluginInput{
		Name: pluginName,
	})
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error reading plugin named %s: %s", pluginName, err))
		return 2
	}

	data := map[string]interface{}{
		"args":    resp.Args,
		"builtin": resp.Builtin,
		"command": resp.Command,
		"name":    resp.Name,
		"sha256":  resp.SHA256,
	}

	if c.flagField != "" {
		return PrintRawField(c.UI, data, c.flagField)
	}
	return OutputData(c.UI, data)
}
