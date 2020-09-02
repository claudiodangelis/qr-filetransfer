package cmd

import (
	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/qr"
	log "github.com/sirupsen/logrus"

	"github.com/claudiodangelis/qrcp/server"
	"github.com/spf13/cobra"
)

func sendCmdFunc(command *cobra.Command, args []string) error {
	payload, err := payload.FromArgs(args, zipFlag)
	if err != nil {
		log.Error(err)
	}
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
	srv, err := server.New(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	// Sets the payload
	srv.Send(payload)
	log.WithFields(log.Fields{
		"File(s) Selected": args,
	}).Info("Scan the following URL with a QR reader to start the file transfer:")
	log.Print(srv.SendURL)
	qr.RenderString(srv.SendURL)
	if browserFlag {
		srv.DisplayQR(srv.SendURL)
	}

	if err := srv.Wait(); err != nil {
		log.Fatal(err)

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
