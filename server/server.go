package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/claudiodangelis/qrcp/qr"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/pages"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/util"
	"gopkg.in/cheggaaa/pb.v1"
)

// Server is the server
type Server struct {
	BaseURL string
	// SendURL is the URL used to send the file
	SendURL string
	// ReceiveURL is the URL used to Receive the file
	ReceiveURL  string
	instance    *http.Server
	payload     payload.Payload
	outputDir   string
	stopChannel chan bool
	// expectParallelRequests is set to true when qrcp sends files, in order
	// to support downloading of parallel chunks
	expectParallelRequests bool
}

// ReceiveTo sets the output directory
func (s *Server) ReceiveTo(dir string) error {
	output, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	// Check if the output dir exists
	fileinfo, err := os.Stat(output)
	if err != nil {
		return err
	}
	if !fileinfo.IsDir() {
		return fmt.Errorf("%s is not a valid directory", output)
	}
	s.outputDir = output
	return nil
}

// Send adds a handler for sending the file
func (s *Server) Send(p payload.Payload) {
	s.payload = p
	s.expectParallelRequests = true
}

// DisplayQR creates a handler for serving the QR code in the browser
func (s *Server) DisplayQR(url string) {
	const PATH = "/qr"
	qrImg := qr.RenderImage(url)
	http.HandleFunc(PATH, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, qrImg, nil)
	})
	openBrowser(s.BaseURL + PATH)
}

// Wait for transfer to be completed, it waits forever if kept awlive
func (s Server) Wait() error {
	<-s.stopChannel
	if err := s.instance.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
	if s.payload.DeleteAfterTransfer {
		s.payload.Delete()
	}
	return nil
}

// Shutdown the server
func (s Server) Shutdown() {
	s.stopChannel <- true
}

// New instance of the server
func New(cfg *config.Config) (*Server, error) {

	app := &Server{}
	// Get the address of the configured interface to bind the server to
	bind, err := util.GetInterfaceAddress(cfg.Interface)
	if err != nil {
		return &Server{}, err
	}
	// Create a listener. If `port: 0`, a random one is chosen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, cfg.Port))
	if err != nil {
		return nil, err
	}
	// Set the value of computed port
	port := listener.Addr().(*net.TCPAddr).Port
	// Set the host
	host := fmt.Sprintf("%s:%d", bind, port)
	// Get a random path to use
	path := cfg.Path
	if path == "" {
		path = util.GetRandomURLPath()
	}
	// Set the hostname
	hostname := fmt.Sprintf("%s:%d", bind, port)
	// Use external IP when using `interface: any`, unless a FQDN is set
	if bind == "0.0.0.0" && cfg.FQDN == "" {
		fmt.Println("Retrieving the external IP...")
		extIP, err := util.GetExernalIP()
		if err != nil {
			panic(err)
		}
		hostname = fmt.Sprintf("%s:%d", extIP.String(), port)
	}
	// Use a fully-qualified domain name if set
	if cfg.FQDN != "" {
		hostname = fmt.Sprintf("%s:%d", cfg.FQDN, port)
	}
	// Set URLs
	protocol := "http"
	if cfg.Secure {
		protocol = "https"
	}
	app.BaseURL = fmt.Sprintf("%s://%s", protocol, hostname)
	app.SendURL = fmt.Sprintf("%s/send/%s",
		app.BaseURL, path)
	app.ReceiveURL = fmt.Sprintf("%s/receive/%s",
		app.BaseURL, path)
	// Create a server
	httpserver := &http.Server{
		Addr: host,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	// Create channel to send message to stop server
	app.stopChannel = make(chan bool)
	// Create cookie used to verify request is coming from first client to connect
	cookie := http.Cookie{Name: "qrcp", Value: ""}
	// Gracefully shutdown when an OS signal is received or when "q" is pressed
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		app.stopChannel <- true
	}()
	// The handler adds and removes from the sync.WaitGroup
	// When the group is zero all requests are completed
	// and the server is shutdown
	var waitgroup sync.WaitGroup
	waitgroup.Add(1)
	var initCookie sync.Once
	// Create handlers
	// Send handler (sends file to caller)
	http.HandleFunc("/send/"+path, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("User-Agent"), "Mozilla") {
			if cookie.Value == "" {
				initCookie.Do(func() {
					value, err := util.GetSessionID()
					if err != nil {
						log.Println("Unable to generate session ID", err)
						app.stopChannel <- true
						return
					}
					cookie.Value = value
					http.SetCookie(w, &cookie)
				})
			} else {
				// Check for the expected cookie and value
				// If it is missing or doesn't match
				// return a 404 status
				rcookie, err := r.Cookie(cookie.Name)
				if err != nil || rcookie.Value != cookie.Value {
					http.Error(w, "", http.StatusNotFound)
					return
				}
				// If the cookie exits and matches
				// this is an aadditional request.
				// Increment the waitgroup
				waitgroup.Add(1)
			}
		}
		// Remove connection from the waitgroup when done
		defer waitgroup.Done()
		w.Header().Set("Content-Disposition", "attachment; filename="+
			app.payload.Filename)
		http.ServeFile(w, r, app.payload.Path)
	})
	// Upload handler (serves the upload page)
	http.HandleFunc("/receive/"+path, func(w http.ResponseWriter, r *http.Request) {
		htmlVariables := struct {
			Route string
			File  string
		}{}
		htmlVariables.Route = "/receive/" + path
		switch r.Method {
		case "POST":
			filenames := util.ReadFilenames(app.outputDir)
			reader, err := r.MultipartReader()
			if err != nil {
				fmt.Fprintf(w, "Upload error: %v\n", err)
				log.Printf("Upload error: %v\n", err)
				app.stopChannel <- true
				return
			}
			transferredFiles := []string{}
			progressBar := pb.New64(r.ContentLength)
			progressBar.ShowCounters = false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				// iIf part.FileName() is empty, skip this iteration.
				if part.FileName() == "" {
					continue
				}
				// Prepare the destination
				fileName := getFileName(sanitizeFileName(part.FileName()), filenames)
				out, err := os.Create(filepath.Join(app.outputDir, fileName))
				if err != nil {
					// Output to server
					fmt.Fprintf(w, "Unable to create the file for writing: %s\n", err)
					// Output to console
					log.Printf("Unable to create the file for writing: %s\n", err)
					// Send signal to server to shutdown
					app.stopChannel <- true
					return
				}
				defer out.Close()
				// Add name of new file
				filenames = append(filenames, fileName)
				// Write the content from POSTed file to the out
				fmt.Println("Transferring file: ", out.Name())
				progressBar.Prefix(out.Name())
				progressBar.Start()
				buf := make([]byte, 1024)
				for {
					// Read a chunk
					n, err := part.Read(buf)
					if err != nil && err != io.EOF {
						// Output to server
						fmt.Fprintf(w, "Unable to write file to disk: %v", err)
						// Output to console
						fmt.Printf("Unable to write file to disk: %v", err)
						// Send signal to server to shutdown
						app.stopChannel <- true
						return
					}
					if n == 0 {
						break
					}
					// Write a chunk
					if _, err := out.Write(buf[:n]); err != nil {
						// Output to server
						fmt.Fprintf(w, "Unable to write file to disk: %v", err)
						// Output to console
						log.Printf("Unable to write file to disk: %v", err)
						// Send signal to server to shutdown
						app.stopChannel <- true
						return
					}
					progressBar.Add(n)
				}
				transferredFiles = append(transferredFiles, out.Name())
			}
			progressBar.FinishPrint("File transfer completed")
			// Set the value of the variable to the actually transferred files
			htmlVariables.File = strings.Join(transferredFiles, ", ")
			serveTemplate("done", pages.Done, w, htmlVariables)
			if cfg.KeepAlive == false {
				app.stopChannel <- true
			}
		case "GET":
			serveTemplate("upload", pages.Upload, w, htmlVariables)
		}
	})
	// Wait for all wg to be done, then send shutdown signal
	go func() {
		waitgroup.Wait()
		if cfg.KeepAlive || !app.expectParallelRequests {
			return
		}
		app.stopChannel <- true
	}()
	go func() {
		netListener := tcpKeepAliveListener{listener.(*net.TCPListener)}
		if cfg.Secure {
			if err := httpserver.ServeTLS(netListener, cfg.TLSCert, cfg.TLSKey); err != http.ErrServerClosed {
				log.Fatalln("error starting the server:", err)
			}
		} else {
			if err := httpserver.Serve(netListener); err != http.ErrServerClosed {
				log.Fatalln("error starting the server", err)
			}
		}
	}()
	app.instance = httpserver
	return app, nil
}

// openBrowser navigates to a url using the default system browser
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("failed to open browser on platform: %s", runtime.GOOS)
	}
	if err != nil {
		log.Fatal(err)
	}
}
