package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

var (
	rri = map[string]int{}
)

func handler(w http.ResponseWriter, r *http.Request) {
	// First ensure HTTPS if we have a proper cert
	if tlsConfig.NameToCertificate != nil {
		httpsifyLock.RLock()
		st, ok := httpsify[r.Host]
		httpsifyLock.RUnlock()
		if ok {
			if st {
				http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
				return
			}
		} else {
			name := strings.ToLower(r.Host)
			for len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}

			if _, ok := tlsConfig.NameToCertificate[name]; ok {
				httpsifyLock.Lock()
				httpsify[r.Host] = true
				httpsifyLock.Unlock()
				http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
				return
			}

			labels := strings.Split(name, ".")
			for i := range labels {
				labels[i] = "*"
				candidate := strings.Join(labels, ".")
				if _, ok := tlsConfig.NameToCertificate[candidate]; ok {
					httpsifyLock.Lock()
					httpsify[r.Host] = true
					httpsifyLock.Unlock()
					http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
					return
				}
			}

			httpsifyLock.Lock()
			httpsify[r.Host] = false
			httpsifyLock.Unlock()
		}
	}

	backend, ok := domains[r.Host]
	if !ok {
		w.Write([]byte("No route found"))
		return
	}

	targets, ok := backends[backend]
	if !ok {
		w.Write([]byte("Invalid route"))
		return
	}

	index := 0
	if rr, ok := rri[backend]; ok {
		index = rr
		if rr > len(targets)-1 {
			rr = 0
		}
	} else {
		rri[backend] = 0
		index = 0
	}

	r.RequestURI = ""
	r.URL.Scheme = "http"

	remote_addr := r.RemoteAddr
	idx := strings.LastIndex(remote_addr, ":")
	if idx != -1 {
		remote_addr = remote_addr[0:idx]
		if remote_addr[0] == '[' && remote_addr[len(remote_addr)-1] == ']' {
			remote_addr = remote_addr[1 : len(remote_addr)-1]
		}
	}
	r.Header.Add("X-Forwarded-For", remote_addr)

	r.URL.Host = targets[index]

	conn_hdr := ""
	conn_hdrs := r.Header["Connection"]
	if len(conn_hdrs) > 0 {
		conn_hdr = conn_hdrs[0]
	}

	upgrade_websocket := false
	if strings.ToLower(conn_hdr) == "upgrade" {
		upgrade_hdrs := r.Header["Upgrade"]
		if len(upgrade_hdrs) > 0 {
			upgrade_websocket = (strings.ToLower(upgrade_hdrs[0]) == "websocket")
		}
	}

	if upgrade_websocket {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}

		conn, bufrw, err := hj.Hijack()
		defer conn.Close()

		conn2, err := net.Dial("tcp", targets[index])
		if err != nil {
			http.Error(w, "couldn't connect to backend server", http.StatusServiceUnavailable)
			return
		}
		defer conn2.Close()

		err = r.Write(conn2)
		if err != nil {
			return
		}

		CopyBidir(conn, bufrw, conn2, bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2)))
	} else {
		transport := &http.Transport{DisableKeepAlives: false, DisableCompression: false}

		resp, err := transport.RoundTrip(r)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}

		w.WriteHeader(resp.StatusCode)

		io.Copy(w, resp.Body)
		resp.Body.Close()
	}
}
