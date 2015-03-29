package main

import (
	"crypto/tls"
	"strings"

	"github.com/Sirupsen/logrus"
)

var (
	certs    []tls.Certificate
	domains  map[string]string
	backends map[string][]string
)

func loadSettings() {
	// Reset the storage
	newCerts := []tls.Certificate{}
	newDomains := map[string]string{}
	newBackends := map[string][]string{}

	// Fetch config
	resp, err := etcc.Get(*etcdPath, true, true)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the fetched node
	for _, node := range resp.Node.Nodes {
		// Remove the prefix
		key := node.Key[len(*etcdPath)+1:]

		switch key {
		case "certs":
			// Parse the pairs
			for _, pair := range node.Nodes {
				// Remove the prefix
				domain := pair.Key[len(*etcdPath)+7:]

				var (
					cert string
					key  string
				)

				// Parse each part
				for _, part := range pair.Nodes {
					kind := part.Key[len(*etcdPath)+len(domain)+8:]

					if kind == "cert" {
						cert = part.Value
					} else if kind == "key" {
						key = part.Value
					}
				}

				// Parse the pair
				keypair, err := tls.X509KeyPair([]byte(cert), []byte(key))
				if err != nil {
					log.WithFields(logrus.Fields{
						"domain": domain,
						"error":  err.Error(),
					}).Error("Unable to load a key pair")
					continue
				}

				// Append it to the certificates slice
				newCerts = append(newCerts, keypair)

				// Log some debug information
				log.WithFields(logrus.Fields{
					"domain": domain,
				}).Info("Loaded a SSL key pair")
			}
		case "domains":
			// Loop over domains
			for _, domain := range node.Nodes {
				// Remove the prefix
				key := domain.Key[len(*etcdPath)+9:]
				newDomains[key] = domain.Value

				// Log some debug information
				log.WithFields(logrus.Fields{
					"domain":  key,
					"backend": domain.Value,
				}).Info("Added a new domain mapping")
			}
		case "backends":
			// Loop over backends
			for _, backend := range node.Nodes {
				// Remove the prefix
				key := backend.Key[len(*etcdPath)+10:]
				newBackends[key] = strings.Split(backend.Value, "\n")

				// Log some debug information
				log.WithFields(logrus.Fields{
					"backend": key,
				}).Info("Added a new backend")
			}
		}
	}

	certs = newCerts

	tlsConfig.Certificates = certs
	tlsConfig.BuildNameToCertificate()

	domains = newDomains
	backends = newBackends
}
