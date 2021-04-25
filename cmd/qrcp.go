package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	// Global command flags
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "only print errors")
	rootCmd.PersistentFlags().BoolVarP(&keepaliveFlag, "keep-alive", "k", false, "keep server alive after transferring")
	rootCmd.PersistentFlags().BoolVarP(&listallinterfacesFlag, "list-all-interfaces", "l", false, "list all available interfaces when choosing the one to use")
	rootCmd.PersistentFlags().IntVarP(&portFlag, "port", "p", 0, "port to use for the server")
	rootCmd.PersistentFlags().StringVar(&pathFlag, "path", "", "path to use. Defaults to a random string")
	rootCmd.PersistentFlags().StringVarP(&interfaceFlag, "interface", "i", "", "network interface to use for the server")
	rootCmd.PersistentFlags().StringVarP(&fqdnFlag, "fqdn", "d", "", "fully-qualified domain name to use for the resulting URLs")
	rootCmd.PersistentFlags().StringVarP(&hostportFlag, "hostport", "D", "", "host and port to accept really FQDN")
	rootCmd.PersistentFlags().BoolVarP(&zipFlag, "zip", "z", false, "zip content before transferring")
	rootCmd.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "path to the config file, defaults to $XDG_CONFIG_HOME/qrcp/config.json")
	rootCmd.PersistentFlags().BoolVarP(&browserFlag, "browser", "b", false, "display the QR code in a browser window")
	rootCmd.PersistentFlags().BoolVarP(&secureFlag, "secure", "s", false, "use https connection")
	rootCmd.PersistentFlags().StringVar(&tlscertFlag, "tls-cert", "", "path to TLS certificate to use with HTTPS")
	rootCmd.PersistentFlags().StringVar(&tlskeyFlag, "tls-key", "", "path to TLS private key to use with HTTPS")
	// Receive command flags
	receiveCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "output directory for receiving files")
}

// Flags
var zipFlag bool
var portFlag int
var interfaceFlag string
var outputFlag string
var keepaliveFlag bool
var quietFlag bool
var fqdnFlag string
var hostportFlag string
var pathFlag string
var listallinterfacesFlag bool
var configFlag string
var browserFlag bool
var secureFlag bool
var tlscertFlag string
var tlskeyFlag string

// The root command (`qrcp`) is like a shortcut of the `send` command
var rootCmd = &cobra.Command{
	Use:           "qrcp",
	Args:          cobra.MinimumNArgs(1),
	RunE:          sendCmdFunc,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		rootCmd.PrintErrf("Error: %v\nRun `qrcp help` for help.\n", err)
		return err
	}
	return nil
}
