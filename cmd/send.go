package cmd

import (
	"fmt"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/logger"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/qr"
	"github.com/eiannone/keyboard"

	"github.com/claudiodangelis/qrcp/server"
	"github.com/spf13/cobra"
)

func sendCmdFunc(command *cobra.Command, args []string) error {
	log := logger.New(quietFlag)
	payload, err := payload.FromArgs(args, zipFlag)
	if err != nil {
		return err
	}
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
	}
	cfg, err := config.New(configFlag, configOptions)
	if err != nil {
		return err
	}
	srv, err := server.New(&cfg)
	if err != nil {
		return err
	}
	// Sets the payload
	srv.Send(payload)
	log.Print(`Scan the following URL with a QR reader to start the file transfer, press CTRL+C or "q" to exit:`)
	log.Print(srv.SendURL)
	qr.RenderString(srv.SendURL)
	if browserFlag {
		srv.DisplayQR(srv.SendURL)
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

var sendCmd = &cobra.Command{
	Use:     "send",
	Short:   "Send a file(s) or directories from this host",
	Long:    "Send a file(s) or directories from this host",
	Aliases: []string{"s"},
	Example: `# Send /path/file.gif. Webserver listens on a random port
qrcp send /path/file.gif
# Shorter version:
qrcp /path/file.gif
# Zip file1.gif and file2.gif, then send the zip package
qrcp /path/file1.gif /path/file2.gif
# Zip the content of directory, then send the zip package
qrcp /path/directory
# Send file.gif by creating a webserver on port 8080
qrcp --port 8080 /path/file.gif
`,
	Args: cobra.MinimumNArgs(1),
	RunE: sendCmdFunc,
}
