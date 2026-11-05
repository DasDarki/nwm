package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/DasDarki/nwm/internal/server"
	"github.com/spf13/cobra"
)

func newServerLogsCommand() *cobra.Command {
	var follow bool
	var lines int
	command := &cobra.Command{
		Use:               "logs [name]",
		Short:             "Show the output of a server started with --detach",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			file, err := os.Open(server.LogFile(target.Name))
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("no log for %q yet, logs are written when started with --detach", target.Name)
			}
			if err != nil {
				return err
			}
			defer file.Close()
			if err := printTail(file, lines); err != nil {
				return err
			}
			if !follow {
				return nil
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()
			return followFile(ctx, file)
		},
	}
	command.Flags().BoolVarP(&follow, "follow", "f", false, "keep printing new output")
	command.Flags().IntVarP(&lines, "lines", "n", 50, "number of lines to show, 0 for all")
	return command
}

func printTail(file *os.File, count int) error {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var buffer []string
	for scanner.Scan() {
		buffer = append(buffer, scanner.Text())
		if count > 0 && len(buffer) > count {
			buffer = buffer[1:]
		}
	}
	for _, line := range buffer {
		fmt.Println(line)
	}
	return scanner.Err()
}

func followFile(ctx context.Context, file *os.File) error {
	chunk := make([]byte, 32*1024)
	for {
		read, err := file.Read(chunk)
		if read > 0 {
			os.Stdout.Write(chunk[:read])
			continue
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(250 * time.Millisecond):
		}
	}
}
