package util

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/jhoonb/archivex"
)

// ZipFiles and return the resulting zip's filename
func ZipFiles(files []string) (string, error) {
	zip := new(archivex.ZipFile)
	tmpfile, err := ioutil.TempFile("", "qrcp")
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	if err := os.Rename(tmpfile.Name(), tmpfile.Name()+".zip"); err != nil {
		return "", err
	}
	zip.Create(tmpfile.Name() + ".zip")
	for _, filename := range files {
		fileinfo, err := os.Stat(filename)
		if err != nil {
			return "", err
		}
		if fileinfo.IsDir() {
			zip.AddAll(filename, true)
		} else {
			file, err := os.Open(filename)
			if err != nil {
				return "", err
			}
			defer file.Close()
			if err := zip.Add(filename, file, fileinfo); err != nil {
				return "", err
			}
		}
	}
	if err := zip.Close(); err != nil {
		return "", nil
	}
	return zip.Name, nil
}

// GetRandomURLPath returns a random string of 4 alphanumeric characters
func GetRandomURLPath() string {
	timeNum := time.Now().UTC().UnixNano()
	alphaString := strconv.FormatInt(timeNum, 36)
	return alphaString[len(alphaString)-4:]
}

// GetSessionID returns a base64 encoded string of 40 random characters
func GetSessionID() (string, error) {
	randbytes := make([]byte, 40)
	if _, err := io.ReadFull(rand.Reader, randbytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(randbytes), nil
}

// GetInterfaceAddress returns the address of the network interface to
// bind the server to. If the interface is "any", it will return 0.0.0.0.
// If no interface is found with that name, an error is returned
func GetInterfaceAddress(ifaceString string) (string, error) {
	if ifaceString == "any" {
		return "0.0.0.0", nil
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	var candidateInterface *net.Interface
	for _, iface := range ifaces {
		if iface.Name == ifaceString {
			candidateInterface = &iface
			break
		}
	}
	if candidateInterface != nil {
		ip, err := FindIP(*candidateInterface)
		if err != nil {
			return "", err
		}
		return ip, nil
	}
	return "", errors.New("unable to find interface")
}

// FindIP returns the IP address of the passed interface, and an error
func FindIP(iface net.Interface) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
			return "[" + ipnet.IP.String() + "]", nil
		}
	}
	return "", errors.New("Unable to find an IP for this interface")
}

func filterInterfaces(ifaces []net.Interface) []net.Interface {
	filtered := []net.Interface{}
	var re = regexp.MustCompile(`^(veth|br\-|docker|lo|EHC|XHC|bridge|gif|stf|p2p|awdl|utun|tun|tap)`)
	for _, iface := range ifaces {
		if re.MatchString(iface.Name) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		filtered = append(filtered, iface)
	}
	return filtered
}

// ReadFilenames from dir
func ReadFilenames(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	// Create array of names of files which are stored in dir
	// used later to set valid name for received files
	filenames := make([]string, 0, len(files))
	for _, fi := range files {
		// ignore empty filename
		fName := fi.Name()
		if len(fName) == 0 {
			continue
		}
		filenames = append(filenames, fName)
	}
	return filenames
}
