package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/namsral/flag"
)

var (
	configFlag       = flag.String("config", "", "Config file to read")                  // Enable config file cuntionality
	logFormatterType = flag.String("log_formatter_type", "text", "Log formatter to use") // Logrus log formatter
	logForceColors   = flag.Bool("log_force_colors", false, "Force colored log output?") // Logrus force colors
	etcdAddress      = flag.String("etcd_address", "", "Address of the etcd server to use")
	etcdPath         = flag.String("etcd_path", "", "Prefix of the etcd directory, no slash near the end")

	rawBind = flag.String("raw_bind", "0.0.0.0:80", "Address used for the HTTP server")
	tlsBind = flag.String("tls_bind", "0.0.0.0:443", "Address used for the HTTPS server")
)

var (
	tlsConfig *tls.Config
)

func main() {
	// Parse the flags
	flag.Parse()

	// Normalize the etcd path
	if (*etcdPath)[0] != '/' {
		*etcdPath = "/" + *etcdPath
	}
	if (*etcdPath)[len(*etcdPath)-1] == '/' {
		*etcdPath = (*etcdPath)[:len(*etcdPath)-2]
	}

	// Transform etcdAddress into addresses
	addresses := strings.Split(*etcdAddress, ",")
	etcc = etcd.NewClient(addresses)

	// Set up a new logger
	log = logrus.New()

	// Set the formatter depending on the passed flag's value
	if *logFormatterType == "text" {
		log.Formatter = &logrus.TextFormatter{
			ForceColors: *logForceColors,
		}
	} else if *logFormatterType == "json" {
		log.Formatter = &logrus.JSONFormatter{}
	}

	// Create a new TLS config
	tlsConfig = &tls.Config{
		Certificates: certs,
		MinVersion:   tls.VersionTLS10,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		},
		PreferServerCipherSuites: true,
	}
	tlsConfig.BuildNameToCertificate()

	// Load the settings
	loadSettings()

	// Start the HTTPS server
	go func() {
		conn, err := net.Listen("tcp", *tlsBind)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error":   err.Error(),
				"address": *tlsBind,
			}).Fatal("Unable to prepare a listener for the HTTPS server")
		}

		listener := tls.NewListener(conn, tlsConfig)

		server := &http.Server{
			Handler: http.HandlerFunc(handler),
		}

		err = server.Serve(listener)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Fatal("Error while serving a HTTPS listener")
		}
	}()

	// Start the HTTP server
	go func() {
		err := http.ListenAndServe(*rawBind, http.HandlerFunc(handler))
		if err != nil {
			log.WithFields(logrus.Fields{
				"error":   err.Error(),
				"address": *rawBind,
			}).Fatal("Unable to bind the HTTP server")
		}
	}()

	// Watch for changes in etcd
	receiver := make(chan *etcd.Response)
	stop := make(chan bool)

	go func() {
		for {
			// Wait for a change
			<-receiver

			// Log debug information
			log.Info("Reloading settings")

			// Reload the settings
			loadSettings()
		}
	}()

	_, err := etcc.Watch(*etcdPath, 0, true, receiver, stop)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("etcd watcher errored")
	}
}
