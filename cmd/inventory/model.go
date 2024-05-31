package main

import (
	"fmt"
	"sort"

	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// NewModelCommand returns a new command for interfacing with the models.
func NewModelCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "model",
		Usage:   "model operations",
		Aliases: []string{"m"},
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Usage:   "list registered models",
				Aliases: []string{"ls"},
				Action: func(ctx *cli.Context) error {
					models := make([]string, 0, registry.ModelRegistry.Length())
					walker := func(name string, _ any) error {
						models = append(models, name)
						return nil
					}

					if err := registry.ModelRegistry.Range(walker); err != nil {
						return err
					}

					sort.Strings(models)
					for _, model := range models {
						fmt.Println(model)
					}

					return nil
				},
			},
		},
	}

	return cmd
}
