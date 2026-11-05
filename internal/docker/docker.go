package docker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/ui"
)

type Container struct {
	Name    string
	Service string
	State   string
	Status  string
	Image   string
	Ports   string
}

func (c Container) Running() bool {
	return c.State == "running"
}

func Available(ctx context.Context) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker is not installed, services need Docker (https://docs.docker.com/get-docker/)")
	}
	output, err := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker daemon is not reachable: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func Up(ctx context.Context, server string, directory string, service Service, report ui.Reporter) error {
	if err := service.Validate(); err != nil {
		return err
	}
	name := ContainerName(server, service.Name)
	running, hash, exists := inspect(ctx, name)
	if exists && hash != service.hash() {
		report.Status("definition changed, recreating " + name)
		if err := run(ctx, "rm", "--force", name); err != nil {
			return err
		}
		exists = false
	}
	if exists {
		if !running {
			report.Status("starting " + name)
			if err := run(ctx, "start", name); err != nil {
				return err
			}
		}
	} else {
		if err := pull(ctx, service.Image, report); err != nil {
			return err
		}
		report.Status("creating " + name)
		if err := run(ctx, runArguments(server, directory, service)...); err != nil {
			return err
		}
	}
	if service.ShouldWait() {
		return waitForPort(ctx, service, report)
	}
	return nil
}

func Down(ctx context.Context, server string, service string) error {
	name := ContainerName(server, service)
	if _, _, exists := inspect(ctx, name); !exists {
		return nil
	}
	return run(ctx, "stop", name)
}

func Remove(ctx context.Context, server string, service Service, withVolumes bool) error {
	name := ContainerName(server, service.Name)
	if _, _, exists := inspect(ctx, name); exists {
		if err := run(ctx, "rm", "--force", name); err != nil {
			return err
		}
	}
	if !withVolumes {
		return nil
	}
	for _, volume := range NamedVolumes(server, service) {
		exec.CommandContext(ctx, "docker", "volume", "rm", volume).Run()
	}
	return nil
}

func List(ctx context.Context, server string) ([]Container, error) {
	output, err := exec.CommandContext(ctx, "docker", "ps", "--all",
		"--filter", "label=nwm.server="+strings.ToLower(server),
		"--format", `{{.Names}}|{{.Label "nwm.service"}}|{{.State}}|{{.Status}}|{{.Image}}|{{.Ports}}`).Output()
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	var containers []Container
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.SplitN(line, "|", 6)
		if len(fields) < 6 {
			continue
		}
		containers = append(containers, Container{Name: fields[0], Service: fields[1], State: fields[2], Status: fields[3], Image: fields[4], Ports: fields[5]})
	}
	return containers, nil
}

func Logs(ctx context.Context, server string, service string, follow bool, lines int) error {
	arguments := []string{"logs", "--tail", strconv.Itoa(lines)}
	if lines <= 0 {
		arguments[2] = "all"
	}
	if follow {
		arguments = append(arguments, "--follow")
	}
	command := exec.CommandContext(ctx, "docker", append(arguments, ContainerName(server, service))...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func inspect(ctx context.Context, name string) (bool, string, bool) {
	output, err := exec.CommandContext(ctx, "docker", "inspect", "--type", "container",
		"--format", `{{.State.Running}}|{{index .Config.Labels "nwm.hash"}}`, name).Output()
	if err != nil {
		return false, "", false
	}
	running, hash, _ := strings.Cut(strings.TrimSpace(string(output)), "|")
	return running == "true", hash, true
}

func pull(ctx context.Context, image string, report ui.Reporter) error {
	if exec.CommandContext(ctx, "docker", "image", "inspect", image).Run() == nil {
		return nil
	}
	report.Status("pulling " + image)
	return process.Stream(exec.CommandContext(ctx, "docker", "pull", image), report.Status)
}

func run(ctx context.Context, arguments ...string) error {
	return process.Stream(exec.CommandContext(ctx, "docker", arguments...), nil)
}

func waitForPort(ctx context.Context, service Service, report ui.Reporter) error {
	port, err := HostPort(service.Ports[0])
	if err != nil {
		return err
	}
	address := net.JoinHostPort("127.0.0.1", port)
	report.Status("waiting for " + service.Name + " on " + address)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			connection.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("service %q did not open %s within 60s", service.Name, address)
}
