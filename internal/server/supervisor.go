package server

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/DasDarki/nwm/internal/home"
)

const (
	supervisorCommand = "__supervise"
	stopTimeout       = 20 * time.Second
)

func SupervisorCommand() string {
	return supervisorCommand
}

func SocketPath(name string) string {
	sum := sha256.Sum256([]byte(home.Dir() + "/" + strings.ToLower(name)))
	return filepath.Join(os.TempDir(), "nwm-"+hex.EncodeToString(sum[:6])+".sock")
}

type Control struct {
	listener net.Listener
	socket   string
}

func ServeControl(name string, console func() *Console) (*Control, error) {
	socket := SocketPath(name)
	os.Remove(socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("opening control socket: %w", err)
	}
	control := &Control{listener: listener, socket: socket}
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go handleControl(connection, console)
		}
	}()
	return control, nil
}

func (c *Control) Socket() string {
	return c.socket
}

func (c *Control) Close() {
	c.listener.Close()
	os.Remove(c.socket)
}

func Supervise(name string, directory string, arguments []string) error {
	console, err := StartConsole(directory, arguments, nil, func(line string) {
		fmt.Fprintln(os.Stdout, line)
	})
	if err != nil {
		return err
	}
	control, err := ServeControl(name, func() *Console { return console })
	if err != nil {
		console.Stop(stopTimeout)
		return err
	}
	defer control.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		console.Stop(stopTimeout)
	}()

	return console.Err()
}

func handleControl(connection net.Conn, current func() *Console) {
	defer connection.Close()
	scanner := bufio.NewScanner(connection)
	for scanner.Scan() {
		verb, argument, _ := strings.Cut(scanner.Text(), " ")
		console := current()
		if console == nil {
			fmt.Fprintln(connection, "error the server is not running")
			continue
		}
		switch verb {
		case "cmd":
			if err := console.Send(argument); err != nil {
				fmt.Fprintf(connection, "error %v\n", err)
				continue
			}
			fmt.Fprintln(connection, "ok")
		case "stop":
			fmt.Fprintln(connection, "ok")
			console.Stop(stopTimeout)
			return
		default:
			fmt.Fprintf(connection, "error unknown request %q\n", verb)
		}
	}
}

func control(name string, request string) error {
	info, running := Status(name)
	if !running {
		return fmt.Errorf("server %q is not running", name)
	}
	if info.Socket == "" {
		return errors.New("this server was started in the foreground, type commands into its terminal")
	}
	connection, err := net.DialTimeout("unix", info.Socket, 3*time.Second)
	if err != nil {
		return fmt.Errorf("cannot reach the server console: %w", err)
	}
	defer connection.Close()
	connection.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := fmt.Fprintln(connection, request); err != nil {
		return err
	}
	response, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return err
	}
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "error ") {
		return errors.New(strings.TrimPrefix(response, "error "))
	}
	return nil
}

func SendCommand(name string, command string) error {
	return control(name, "cmd "+command)
}
