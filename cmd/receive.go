package cmd

import (
	"fmt"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/logger"
	"github.com/claudiodangelis/qrcp/qr"
	"github.com/claudiodangelis/qrcp/server"
	"github.com/eiannone/keyboard"
	"github.com/spf13/cobra"
)

func receiveCmdFunc(command *cobra.Command, args []string) error {
	log := logger.New(quietFlag)
	// Load configuration
	configOptions := config.Options{
		Interface:         interfaceFlag,
		Port:              portFlag,
		Path:              pathFlag,
		FQDN:              fqdnFlag,
		Hostport:          hostportFlag,
		KeepAlive:         keepaliveFlag,
		ListAllInterfaces: listallinterfacesFlag,
		Secure:            secureFlag,
		TLSCert:           tlscertFlag,
		TLSKey:            tlskeyFlag,
		Output:            outputFlag,
	}
	cfg, err := config.New(configFlag, configOptions)
	if err != nil {
		return err
	}
	// Create the server
	srv, err := server.New(&cfg)
	if err != nil {
		return err
	}
	// Sets the output directory
	if err := srv.ReceiveTo(cfg.Output); err != nil {
		return err
	}
	// Prints the URL to scan to screen
	log.Print(`Scan the following URL with a QR reader to start the file transfer, press CTRL+C or "q" to exit:`)
	log.Print(srv.ReceiveURL)
	// Renders the QR
	qr.RenderString(srv.ReceiveURL)
	if browserFlag {
		srv.DisplayQR(srv.ReceiveURL)
	}
	if err := keyboard.Open(); err == nil {
		defer func() {
			keyboard.Close()
		}()
		go func() {
			for {
				char, key, _ := keyboard.GetKey()
				if string(char) == "q" || key == keyboard.KeyCtrlC {
					srv.Shutdown()
				}
			}
		}()
	} else {
		log.Print(fmt.Sprintf("Warning: keyboard not detected: %v", err))
	}
	if err := srv.Wait(); err != nil {
		return err
	}
	return nil
}

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	Short:   "Receive one or more files",
	Long:    "Receive one or more files. The destination directory can be set with the config wizard, or by passing the --output flag. If none of the above are set, the current working directory will be used as a destination directory.",
	Example: `# Receive files in the current directory
qrcp receive
# Receive files in a specific directory
qrcp receive --output /tmp
`,
	RunE: receiveCmdFunc,
}
