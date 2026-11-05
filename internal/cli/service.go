package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/DasDarki/nwm/internal/docker"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func newServiceCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "service",
		Aliases: []string{"services", "svc"},
		Short:   "🐳 Manage Docker services (databases, caches, ...) next to a server",
		Long: "Services are Docker containers that belong to a server, defined in its nwm.toml.\n" +
			"They start automatically with the server. Ports bind to 127.0.0.1 unless an IP is given.",
	}
	command.AddCommand(
		newServiceListCommand(),
		newServiceAddCommand(),
		newServiceUpCommand(),
		newServiceDownCommand(),
		newServiceLogsCommand(),
		newServiceRemoveCommand(),
	)
	return command
}

func loadServices(serverName string) (*state.Server, *server.Local, error) {
	_, target, err := resolveServer(serverName)
	if err != nil {
		return nil, nil, err
	}
	local, err := server.LoadLocal(target.Path)
	return target, local, err
}

func newServiceListCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List a server's services and their state",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			target, local, err := loadServices(serverName)
			if err != nil {
				return err
			}
			if len(local.Services) == 0 {
				ui.Hint("%s has no services, add one with %s", target.Name, ui.Command("nwm service add"))
				return nil
			}
			states := map[string]docker.Container{}
			if docker.Available(command.Context()) == nil {
				containers, err := docker.List(command.Context(), target.Name)
				if err != nil {
					return err
				}
				for _, container := range containers {
					states[container.Service] = container
				}
			}
			rows := make([][]string, 0, len(local.Services))
			for _, service := range local.Services {
				status := "⚫ not created"
				if container, found := states[service.Name]; found {
					status = "⚫ " + container.Status
					if container.Running() {
						status = "🟢 " + container.Status
					}
				}
				rows = append(rows, []string{service.Name, service.Image, strings.Join(service.Ports, ", "), status})
			}
			fmt.Println(renderTable([]string{"NAME", "IMAGE", "PORTS", "STATUS"}, rows))
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newServiceAddCommand() *cobra.Command {
	var serverName string
	var service docker.Service
	var environment []string
	var start bool
	command := &cobra.Command{
		Use:   "add <name> --image <image>",
		Short: "Add a service to a server",
		Example: "  nwm service add db --image postgres:16 --port 5432 --env POSTGRES_PASSWORD=dev --volume data:/var/lib/postgresql/data\n" +
			"  nwm service add cache --image redis:7-alpine --port 6379",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			target, local, err := loadServices(serverName)
			if err != nil {
				return err
			}
			service.Name = arguments[0]
			service.Env = map[string]string{}
			for _, entry := range environment {
				key, value, found := strings.Cut(entry, "=")
				if !found {
					return fmt.Errorf("--env expects KEY=value, got %q", entry)
				}
				service.Env[key] = value
			}
			if err := service.Validate(); err != nil {
				return err
			}
			if _, exists := local.Service(service.Name); exists {
				ui.Hint("replacing the existing definition of %s", service.Name)
			}
			local.PutService(service)
			if err := local.Save(); err != nil {
				return err
			}
			ui.Success("Added %s to %s", service.Name, target.Name)
			if start {
				return upServices(target, []docker.Service{service})
			}
			return nil
		},
	}
	addServerFlag(command, &serverName)
	command.Flags().StringVar(&service.Image, "image", "", "docker image, e.g. postgres:16")
	command.Flags().StringArrayVarP(&service.Ports, "port", "p", nil, "published port: 5432, 15432:5432 or 0.0.0.0:5432:5432")
	command.Flags().StringArrayVarP(&environment, "env", "e", nil, "environment variable KEY=value")
	command.Flags().StringArrayVarP(&service.Volumes, "volume", "v", nil, "volume: name:/path (managed) or ./dir:/path (relative to the server)")
	command.Flags().BoolVar(&start, "start", true, "start the service right away")
	command.MarkFlagRequired("image")
	return command
}

func upServices(target *state.Server, services []docker.Service) error {
	if len(services) == 0 {
		return nil
	}
	return ui.Run(fmt.Sprintf("Starting %d service(s) for %s", len(services), target.Name), func(ctx context.Context, report ui.Reporter) error {
		if err := docker.Available(ctx); err != nil {
			return err
		}
		for _, service := range services {
			if err := docker.Up(ctx, target.Name, target.Path, service, report); err != nil {
				return err
			}
		}
		return nil
	})
}

func downServices(target *state.Server, services []docker.Service) error {
	if len(services) == 0 {
		return nil
	}
	return ui.Run(fmt.Sprintf("Stopping %d service(s) for %s", len(services), target.Name), func(ctx context.Context, report ui.Reporter) error {
		if err := docker.Available(ctx); err != nil {
			return err
		}
		for _, service := range services {
			report.Status("stopping " + service.Name)
			if err := docker.Down(ctx, target.Name, service.Name); err != nil {
				return err
			}
		}
		return nil
	})
}

func newServiceUpCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "up [service]...",
		Aliases: []string{"start"},
		Short:   "Start services (all if none given)",
		RunE: func(_ *cobra.Command, names []string) error {
			target, local, err := loadServices(serverName)
			if err != nil {
				return err
			}
			services, err := local.SelectServices(names)
			if err != nil {
				return err
			}
			if len(services) == 0 {
				ui.Hint("%s has no services", target.Name)
				return nil
			}
			return upServices(target, services)
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newServiceDownCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "down [service]...",
		Aliases: []string{"stop"},
		Short:   "Stop services (all if none given)",
		RunE: func(_ *cobra.Command, names []string) error {
			target, local, err := loadServices(serverName)
			if err != nil {
				return err
			}
			services, err := local.SelectServices(names)
			if err != nil {
				return err
			}
			return downServices(target, services)
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newServiceLogsCommand() *cobra.Command {
	var serverName string
	var follow bool
	var lines int
	command := &cobra.Command{
		Use:   "logs <service>",
		Short: "Show the output of a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			target, local, err := loadServices(serverName)
			if err != nil {
				return err
			}
			if _, found := local.Service(arguments[0]); !found {
				return fmt.Errorf("unknown service %q", arguments[0])
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()
			return docker.Logs(ctx, target.Name, arguments[0], follow, lines)
		},
	}
	addServerFlag(command, &serverName)
	command.Flags().BoolVarP(&follow, "follow", "f", false, "keep printing new output")
	command.Flags().IntVarP(&lines, "lines", "n", 50, "number of lines to show, 0 for all")
	return command
}

func newServiceRemoveCommand() *cobra.Command {
	var serverName string
	var volumes bool
	command := &cobra.Command{
		Use:     "remove <service>...",
		Aliases: []string{"rm"},
		Short:   "Remove services and their containers",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, names []string) error {
			target, local, err := loadServices(serverName)
			if err != nil {
				return err
			}
			services, err := local.SelectServices(names)
			if err != nil {
				return err
			}
			if err := removeContainers(target, services, volumes); err != nil {
				return err
			}
			for _, service := range services {
				local.RemoveService(service.Name)
			}
			if err := local.Save(); err != nil {
				return err
			}
			ui.Success("Removed %s", strings.Join(names, ", "))
			return nil
		},
	}
	addServerFlag(command, &serverName)
	command.Flags().BoolVar(&volumes, "volumes", false, "also delete the services' named volumes (data loss)")
	return command
}

func removeContainers(target *state.Server, services []docker.Service, volumes bool) error {
	if len(services) == 0 {
		return nil
	}
	return ui.Run("Removing containers", func(ctx context.Context, report ui.Reporter) error {
		if err := docker.Available(ctx); err != nil {
			return err
		}
		for _, service := range services {
			report.Status("removing " + service.Name)
			if err := docker.Remove(ctx, target.Name, service, volumes); err != nil {
				return err
			}
		}
		return nil
	})
}
