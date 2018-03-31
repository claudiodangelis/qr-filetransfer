package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/mdp/qrterminal"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var remoteFlag = flag.Bool("remote", false, "transfer file via file.io")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var debugFlag = flag.Bool("debug", false, "increase verbosity")
var quietFlag = flag.Bool("quiet", false, "ignores non critical output")

func main() {
	flag.Parse()

	// Check how many arguments are passed
	if len(flag.Args()) == 0 {
		log.Fatalln("At least one argument is required")
	}

	content, err := getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	// If the remote flag is specified, upload and generate QR code for remote url
	if *remoteFlag {
		// Upload file to File.io and use that address for connection
		url, err := UploadFile(content)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Hosted at %s\n", url)
		qrterminal.GenerateHalfBlock(url, qrterminal.L, os.Stdout)
		return
	}

	config := LoadConfig()
	if *forceFlag == true {
		config.Delete()
		config = LoadConfig()
	}

	// Get addresses
	address, err := getAddress(&config)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a net.Listener bound to the choosen address on a random port
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:0", address))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// Generate the QR code
	info("Scan the following QR to start the download.")
	info("Make sure that your smartphone is connected to the same WiFi network as this computer.")
  
	qrConfig := qrterminal.Config{
		HalfBlocks:     true,
		Level:          qrterminal.L,
		Writer:         os.Stdout,
		BlackWhiteChar: "\u001b[37m\u001b[40m\u2584\u001b[0m",
		BlackChar:      "\u001b[30m\u001b[40m\u2588\u001b[0m",
		WhiteBlackChar: "\u001b[30m\u001b[47m\u2585\u001b[0m",
		WhiteChar:      "\u001b[37m\u001b[47m\u2588\u001b[0m",
	}
	qrterminal.GenerateWithConfig(fmt.Sprintf("http://%s", listener.Addr().String()), qrConfig)

  // Define a default handler for the requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition",
			"attachment; filename="+content.Name())

		http.ServeFile(w, r, content.Path)
		if content.ShouldBeDeleted {
			if err := content.Delete(); err != nil {
				log.Println("Unable to delete the content from disk", err)
			}
		}
		if err := config.Update(); err != nil {
			log.Println("Unable to update configuration", err)
		}
		os.Exit(0)
	})
	// Start a new server using the listener bound to the choosen address on a random port
	log.Fatalln(http.Serve(listener, nil))
}
