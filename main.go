package main

import(
	"fmt"
	"net/url"
	"net/http"
	"html/template"
	"net/http/httputil"
	"github.com/Sirupsen/logrus"
	"github.com/docopt/docopt.go"
)

/*
TODO: Configuration: set redis DB details
TODO: Redis storage manager
TODO: API endpoints for management functions: AddKey, RevokeKey, UpdateKey, GetKeyDetails, RequestKey (creates a key for user instead of self supplied)
TODO: Secure API endpoints (perhaps with just a shared secret, should be internally used anyway)
TODO: Configuration: Set shared secret
TODO: Configuration: Error template file path
TODO: Add QuotaLimiter so time-based quotas can be added
*/


var log = logrus.New()
var authManager = AuthorisationManager{}
var sessionLimiter = SessionLimiter{}
var config = Config{}
var templates = &template.Template{}
var systemError string = "{\"status\": \"system error, please contact administrator\"}"

func setupGlobals() {
	if config.Storage.Type == "memory" {
		authManager = AuthorisationManager{
			InMemoryStorageManager{
				map[string]string{}}}
	}

	template_file := fmt.Sprintf("%s/error.json", config.TemplatePath)
	templates = template.Must(template.ParseFiles(template_file))
}

func init() {
	usage := `Tyk API Gateway.

	Usage:
		tyk [options]

	Options:
		-h --help      Show this screen
		--conf=FILE    Load a named configuration file
		--test         Create a test key

	`

	arguments, err := docopt.Parse(usage, nil, true, "Tyk v1.0", false)
	if err != nil {
		log.Println("Error while parsing arguments.")
		log.Fatal(err)
	}

	filename := "tyk.conf"
	value, _ := arguments["--conf"]
	if value != nil {
		log.Info(fmt.Sprintf("Using %s for configuration", value.(string)))
		filename = arguments["--conf"].(string)
	} else {
		log.Info("No configuration file defined, will try to use default (./tyk.conf)")
	}

	loadConfig(filename, &config)
	setupGlobals()

	test_value, _ := arguments["--test"].(bool)
	if test_value {
		log.Info("Adding test key: '1234' to storage map")
		authManager.Store.SetKey("1234", "{\"LastCheck\":1399469149,\"Allowance\":5.0,\"Rate\":1.0,\"Per\":1.0}")
	}

}

func main() {
	createSampleSession()
	remote, err := url.Parse(config.TargetUrl)
	if err != nil {
		log.Error("Culdn't parse target URL")
		log.Error(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/tyk/key/", addKeyHandler)
	http.HandleFunc(config.ListenPath, handler(proxy))
	targetPort := fmt.Sprintf(":%d", config.ListenPort)
	err = http.ListenAndServe(targetPort, nil)
	if err != nil {
		log.Error(err)
	}
}


