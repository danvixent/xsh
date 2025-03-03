package main

import (
	"context"
	"log"
	"time"
	_ "time/tzdata"

	"github.com/spf13/cobra"
)

func main() {
	var hosts []string
	var command string
	var keyFile string
	var outputFile string
	var parallelLimit int
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:     "xsh",
		Version: "0.1",
		Short:   "Multi-host ssh command runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			var pl *int
			if parallelLimit > 0 {
				pl = &parallelLimit
			}
			p, err := NewPlan(
				hosts,
				command,
				keyFile,
				outputFile,
				pl,
			)
			if err != nil {
				log.Fatalf("Error creating plan: %s", err)
			}

			err = p.OpenConns()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			result, err := p.Execute(ctx)
			if err != nil {
				return err
			}

			return p.WriteResult(result)
		},
	}

	cmd.PersistentFlags().StringSliceVar(&hosts, "hosts", []string{}, "hosts to connect to")
	cmd.PersistentFlags().StringVar(&command, "command", "", "command to execute")
	cmd.PersistentFlags().StringVar(&keyFile, "key", "", "ssh key file path")
	cmd.PersistentFlags().StringVar(&outputFile, "output", "", "output file path")
	cmd.PersistentFlags().IntVar(&parallelLimit, "parallel-limit", 0, "limit concurrent command execution to specified limit")
	cmd.PersistentFlags().DurationVar(&timeout, "timeout", 2*time.Minute, "timeout for ssh command")
}
