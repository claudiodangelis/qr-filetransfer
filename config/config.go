package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adrg/xdg"
	"github.com/asaskevich/govalidator"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/manifoldco/promptui"
)

// Config of qrcp
type Config struct {
	FQDN      string `json:"fqdn"`
	Hostport  string `json:"hostport"`
	Interface string `json:"interface"`
	Port      int    `json:"port"`
	KeepAlive bool   `json:"keepAlive"`
	Path      string `json:"path"`
	Secure    bool   `json:"secure"`
	TLSKey    string `json:"tls-key"`
	TLSCert   string `json:"tls-cert"`
	Output    string `json:"output"`
}

var configFile string

// Options of the qrcp configuration
type Options struct {
	Interface         string
	Port              int
	Path              string
	FQDN              string
	Hostport          string
	KeepAlive         bool
	Interactive       bool
	ListAllInterfaces bool
	Secure            bool
	TLSCert           string
	TLSKey            string
	Output            string
}

func chooseInterface(opts Options) (string, error) {
	interfaces, err := util.Interfaces(opts.ListAllInterfaces)
	if err != nil {
		return "", err
	}
	if len(interfaces) == 0 {
		return "", errors.New("no interfaces found")
	}

	if len(interfaces) == 1 && opts.Interactive == false {
		for name := range interfaces {
			fmt.Printf("only one interface found: %s, using this one\n", name)
			return name, nil
		}
	}
	// Map for pretty printing
	m := make(map[string]string)
	items := []string{}
	for name, ip := range interfaces {
		label := fmt.Sprintf("%s (%s)", name, ip)
		m[label] = name
		items = append(items, label)
	}
	// Add the "any" interface
	anyIP := "0.0.0.0"
	anyName := "any"
	anyLabel := fmt.Sprintf("%s (%s)", anyName, anyIP)
	m[anyLabel] = anyName
	items = append(items, anyLabel)
	prompt := promptui.Select{
		Items: items,
		Label: "Choose interface",
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return m[result], nil
}

// Load a new configuration
func Load(opts Options) (Config, error) {
	var cfg Config
	// Read the configuration file, if it exists
	if file, err := ioutil.ReadFile(configFile); err == nil {
		// Read the config
		if err := json.Unmarshal(file, &cfg); err != nil {
			return cfg, err
		}
	}
	// Prompt if needed
	if cfg.Interface == "" {
		iface, err := chooseInterface(opts)
		if err != nil {
			return cfg, err
		}
		cfg.Interface = iface
		// Write config
		if err := write(cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// Wizard starts an interactive configuration managements
func Wizard(path string, listAllInterfaces bool) error {
	if err := setConfigFile(path); err != nil {
		return err
	}
	var cfg Config
	if file, err := ioutil.ReadFile(configFile); err == nil {
		// Read the config
		if err := json.Unmarshal(file, &cfg); err != nil {
			return err
		}
	}
	// Ask for interface
	opts := Options{
		Interactive:       true,
		ListAllInterfaces: listAllInterfaces,
	}
	iface, err := chooseInterface(opts)
	if err != nil {
		return err
	}
	cfg.Interface = iface
	// Ask for fully qualified domain name
	validateFqdn := func(input string) error {
		if input != "" && govalidator.IsDNSName(input) == false {
			return errors.New("invalid domain")
		}
		return nil
	}
	promptFqdn := promptui.Prompt{
		Validate: validateFqdn,
		Label:    "Choose fully-qualified domain name",
		Default:  "",
	}
	if promptFqdnString, err := promptFqdn.Run(); err == nil {
		cfg.FQDN = promptFqdnString
	}
	// Ask for port
	validatePort := func(input string) error {
		_, err := strconv.ParseUint(input, 10, 16)
		if err != nil {
			return errors.New("Invalid number")
		}
		return nil
	}

	promptPort := promptui.Prompt{
		Validate: validatePort,
		Label:    "Choose port, 0 means random port",
		Default:  fmt.Sprintf("%d", cfg.Port),
	}
	if promptPortResultString, err := promptPort.Run(); err == nil {
		if port, err := strconv.ParseUint(promptPortResultString, 10, 16); err == nil {
			cfg.Port = int(port)
		}
	}
	validateIsDir := func(input string) error {
		if input == "" {
			return nil
		}
		path, err := filepath.Abs(input)
		if err != nil {
			return err
		}
		f, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !f.IsDir() {
			return errors.New("path is not a directory")
		}
		return nil
	}
	promptOutput := promptui.Prompt{
		Label:    "Choose default output directory for received files, empty does not set a default",
		Default:  cfg.Output,
		Validate: validateIsDir,
	}

	if promptOutputResultString, err := promptOutput.Run(); err == nil {
		if promptOutputResultString != "" {
			p, _ := filepath.Abs(promptOutputResultString)
			cfg.Output = p
		}
	}

	validateHostport := func(input string) error {
		if input != "" && govalidator.IsDialString(input) == false {
			return errors.New("invalid host and port")
		}
		return nil
	}
	promptHostport := promptui.Prompt{
		Validate: validateHostport,
		Label:    "Choose replace host and port",
		Default:  "",
	}
	if promptHostportResultString, err := promptHostport.Run(); err == nil {
		cfg.Hostport = promptHostportResultString
	}

	// Ask for path
	promptPath := promptui.Prompt{
		Label:   "Choose path, empty means random",
		Default: cfg.Path,
	}
	if promptPathResultString, err := promptPath.Run(); err == nil {
		if promptPathResultString != "" {
			cfg.Path = promptPathResultString
		}
	}

	// Ask for keep alive
	promptKeepAlive := promptui.Select{
		Items: []string{"No", "Yes"},
		Label: "Should the server keep alive after transferring?",
	}
	if _, promptKeepAliveResultString, err := promptKeepAlive.Run(); err == nil {
		if promptKeepAliveResultString == "Yes" {
			cfg.KeepAlive = true
		} else {
			cfg.KeepAlive = false
		}

	}
	// TLS
	promptSecure := promptui.Select{
		Items: []string{"No", "Yes"},
		Label: "Should files be securely transferred with HTTPS?",
	}
	if _, promptSecureResultString, err := promptSecure.Run(); err == nil {
		if promptSecureResultString == "Yes" {
			cfg.Secure = true
		} else {
			cfg.Secure = false
		}
	}
	pathIsReadable := func(input string) error {
		if input == "" {
			return nil
		}
		path, err := filepath.Abs(util.Expand(input))
		if err != nil {
			return err
		}
		fmt.Println(path)
		fileinfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fileinfo.Mode().IsDir() {
			return fmt.Errorf(fmt.Sprintf("%s is a directory", input))
		}
		return nil
	}
	// TLS Cert
	promptTLSCert := promptui.Prompt{
		Label:    "Choose TLS certificate path. Empty if not using HTTPS.",
		Default:  cfg.TLSCert,
		Validate: pathIsReadable,
	}
	if promptTLSCertString, err := promptTLSCert.Run(); err == nil {
		cfg.TLSCert = util.Expand(promptTLSCertString)
	}
	// TLS key
	promptTLSKey := promptui.Prompt{
		Label:    "Choose TLS certificate key. Empty if not using HTTPS.",
		Default:  cfg.TLSKey,
		Validate: pathIsReadable,
	}
	if promptTLSKeyString, err := promptTLSKey.Run(); err == nil {
		cfg.TLSKey = util.Expand(promptTLSKeyString)
	}
	// Write it down
	if err := write(cfg); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Configuration updated:\n%s\n", string(b))
	return nil
}

// write the configuration file to disk
func write(cfg Config) error {
	j, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(configFile, j, 0644); err != nil {
		return err
	}
	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func setConfigFile(path string) error {
	// If not explicitly set then use the default
	if path == "" {
		// First try legacy location
		var legacyConfigFile = filepath.Join(xdg.Home, ".qrcp.json")
		if pathExists(legacyConfigFile) {
			configFile = legacyConfigFile
			return nil
		}

		// Else use modern location, first ensuring that the directory
		// exists
		var configDir = filepath.Join(xdg.ConfigHome, "qrcp")
		if !pathExists(configDir) {
			os.Mkdir(configDir, 0744)
		}
		configFile = filepath.Join(configDir, "config.json")
		return nil
	}
	absolutepath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	fileinfo, err := os.Stat(absolutepath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if fileinfo != nil && fileinfo.IsDir() {
		return fmt.Errorf("%s is not a file", absolutepath)
	}
	configFile = absolutepath
	return nil
}

// New returns a new configuration struct. It loads defaults, then overrides
// values if any.
func New(path string, opts Options) (Config, error) {
	var cfg Config
	// Set configFile
	if err := setConfigFile(path); err != nil {
		return cfg, err
	}
	// Load saved file / defaults
	cfg, err := Load(opts)
	if err != nil {
		return cfg, err
	}
	if opts.Interface != "" {
		cfg.Interface = opts.Interface
	}
	if opts.FQDN != "" {
		if govalidator.IsDNSName(opts.FQDN) == false {
			return cfg, errors.New("invalid value for fully-qualified domain name")
		}
		cfg.FQDN = opts.FQDN
	}
	if opts.Port != 0 {
		if opts.Port > 65535 {
			return cfg, errors.New("invalid value for port")
		}
		cfg.Port = opts.Port
	}
	if opts.KeepAlive {
		cfg.KeepAlive = true
	}
	if opts.Path != "" {
		cfg.Path = opts.Path
	}
	if opts.Secure {
		cfg.Secure = true
	}
	if opts.TLSCert != "" {
		cfg.TLSCert = opts.TLSCert
	}
	if opts.TLSKey != "" {
		cfg.TLSKey = opts.TLSKey
	}
	if opts.Output != "" {
		cfg.Output = opts.Output
	}
	return cfg, nil
}
