package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mdp/qrterminal"
	"github.com/phayes/freeport"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var debugFlag = flag.Bool("debug", false, "increase verbosity")

func main() {
	flag.Parse()
	config := LoadConfig()
	if *forceFlag == true {
		config.Delete()
		config = LoadConfig()
	}

	// Check how many arguments are passed
	if len(flag.Args()) == 0 {
		log.Fatalln("At least one argument is required")
	}

	// Get addresses
	address, err := getAddress(&config)
	if err != nil {
		log.Fatalln(err)
	}

	// Get a random available port
	port := freeport.GetPort()
	content, err := getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	// Generate the QR code
	fmt.Println("Scan the following QR to start the download.")
	fmt.Println("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	fmt.Println("Alternatively, use below URL if you do not have QR code scanner installed on your phone.")
	fmt.Sprintf("http://%s:%d", address, port)
	qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%s:%d", address, port),
		qrterminal.L, os.Stdout)

	// Define a default handler for the requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition",
			"attachment; filename="+content.Name())

		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
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
	// Start a new server bound to the chosen address on a random port
	log.Fatalln(http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil))

}
