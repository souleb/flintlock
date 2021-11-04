package command

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/weaveworks/flintlock/internal/command/gw"
	"github.com/weaveworks/flintlock/internal/command/run"
	"github.com/weaveworks/flintlock/internal/config"
	"github.com/weaveworks/flintlock/internal/version"
	"github.com/weaveworks/flintlock/pkg/defaults"
	"github.com/weaveworks/flintlock/pkg/log"
)

const usage = `
  __  _  _         _    _               _        _
 / _|| |(_) _ __  | |_ | |  ___    ___ | | __ __| |
| |_ | || || '_ \ | __|| | / _ \  / __|| |/ // _' |
|  _|| || || | | || |_ | || (_) || (__ |   <| (_| |
|_|  |_||_||_| |_| \__||_| \___/  \___||_|\_\\__,_|

Create and manage the lifecycle of MicroVMs, backed by containerd
`

const configName = "config.yaml"

func NewApp() *cli.App {
	cfg := &config.Config{}
	// Append to the default template
	cli.AppHelpTemplate = fmt.Sprintf(`%s

		WEBSITE: https://docs.flintlock.dev/
		
		SUPPORT: https://github.com/weaveworks/flintlock
		
		`, cli.AppHelpTemplate)

	app := cli.NewApp()
	app.Name = "flintlockd"
	app.Usage = usage
	app.Description = `
flintlock is a service for creating and managing the lifecycle of microVMs on a host machine. 
Initially we will be supporting Firecracker.

The primary use case for flintlock is to create microVMs on a bare-metal host where the microVMs 
will be used as nodes in a virtualized Kubernetes cluster. It is an essential part of 
Liquid Metal and will ultimately be driven by Cluster API Provider Microvm (coming soon).

A default configuration is used if located at the default file location. It can also be defined in a flintlocked directory 
that can be set using the XDG Base directory specification, or in the home directory.`
	app.HideVersion = true

	log.AddFlagsToApp(app, &cfg.Logging)
	addCommands(app, cfg)

	app.Action = func(c *cli.Context) error {
		err := cli.ShowAppHelp(c)
		if err != nil {
			return err
		}

		return nil
	}

	app.Before = func(c *cli.Context) error {
		if err := log.Configure(&cfg.Logging); err != nil {
			return fmt.Errorf("configuring logging: %w", err)
		}

		// Initialization from config file
		// get all flags to initialize
		var cmdFlags []cli.Flag
		for _, command := range app.Commands {
			cmdFlags = append(cmdFlags, command.Flags...)
		}

		// Start with the defaults
		if _, err := os.Stat(fmt.Sprintf("%s/%s", defaults.ConfigurationDir, configName)); err == nil {
			err := resolveFlagsFromConfig(c, fmt.Sprintf("%s/%s", defaults.ConfigurationDir, configName), cmdFlags)
			if err != nil {
				return fmt.Errorf("resolving flags from config: %w", err)
			}
		}
		// Now apply the home config file
		xdgCfg := os.Getenv("XDG_CONFIG_HOME")
		if xdgCfg == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolving flags from config: %w", err)
			}
			xdgCfg = home
		}

		if _, err := os.Stat(fmt.Sprintf("%s/flintlockd/%s", xdgCfg, configName)); err == nil {
			err = resolveFlagsFromConfig(c, fmt.Sprintf("%s/flintlockd/%s", xdgCfg, configName), cmdFlags)
			if err != nil {
				return fmt.Errorf("resolving flags from config: %w", err)
			}
		}

		return nil
	}

	return app
}

func resolveFlagsFromConfig(c *cli.Context, file string, flags []cli.Flag) error {
	inputSource, err := altsrc.NewYamlSourceFromFile(file)
	if err != nil {
		return fmt.Errorf("unable to create input source with context: %w", err)
	}

	err = altsrc.ApplyInputSourceValues(c, inputSource, flags)
	if err != nil {
		return fmt.Errorf("unable to apply input source with context: %w", err)
	}

	return nil
}

func addCommands(app *cli.App, cfg *config.Config) {
	runCmd := run.NewCommand(cfg)
	gwCmd := gw.NewCommand(cfg)
	versionCmd := versionCommand()
	app.Commands = append(app.Commands, runCmd, gwCmd, versionCmd)
}

func versionCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "version",
		Usage: "Print the version number of flintlock",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "long",
				Usage: "Print the long version information",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "short",
				Usage: "Print the short version information",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {

			long := c.Bool("long")
			short := c.Bool("short")

			if short {
				fmt.Fprintln(c.App.Writer, version.Version)

				return nil
			}

			if long {
				fmt.Fprintf(
					c.App.Writer,
					"%s\n  Version:    %s\n  CommitHash: %s\n  BuildDate:  %s\n",
					version.PackageName,
					version.Version,
					version.CommitHash,
					version.BuildDate,
				)

				return nil
			}

			fmt.Fprintf(c.App.Writer, "%s %s\n", version.PackageName, version.Version)

			return nil
		},
	}

	return cmd
}
