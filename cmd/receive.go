package cmd

import (
	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/qr"
	"github.com/claudiodangelis/qrcp/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func receiveCmdFunc(command *cobra.Command, args []string) error {
	// Load configuration
	configOptions := config.Options{
		Interface:         interfaceFlag,
		Port:              portFlag,
		Path:              pathFlag,
		FQDN:              fqdnFlag,
		KeepAlive:         keepaliveFlag,
		ListAllInterfaces: listallinterfacesFlag,
	}
	cfg, err := config.New(configFlag, configOptions)
	if err != nil {
		log.Fatal(err)
	}
	// Create the server
	srv, err := server.New(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	// Sets the output directory
	if err := srv.ReceiveTo(outputFlag); err != nil {
		log.Fatal(err)
	}
	// Prints the URL to scan to screen
	log.Info("Scan the following URL with a QR reader to start the file transfer:")
	log.Print(srv.ReceiveURL)
	// Renders the QR
	qr.RenderString(srv.ReceiveURL)
	if browserFlag {
		srv.DisplayQR(srv.ReceiveURL)
	}
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
	return nil
}

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	Short:   "Receive one or more files",
	Long:    "Receive one or more files. If not specified with the --output flag, the current working directory will be used as a destination.",
	Example: `# Receive files in the current directory
qrcp receive
# Receive files in a specific directory
qrcp receive --output /tmp
`,
	RunE: receiveCmdFunc,
}
