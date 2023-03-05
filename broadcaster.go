package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// read environment variables
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		log.Fatal("DOMAIN environment variable not set")
	}

	httpServer := os.Getenv("LISTEN_SERVER")
	if httpServer == "" {
		httpServer = "0.0.0.0"
	}

	httpPort := os.Getenv("LISTEN_PORT")
	if httpPort == "" {
		httpPort = "80"
	}

	// configure logging to stdout and stderr
	log.SetOutput(os.Stdout)
	errLog := log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// create http client with custom transport that resolves IP addresses for the domain
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := net.Dialer{
				Timeout: 5 * time.Second,
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	client := &http.Client{Transport: tr}

	// create http server that forwards requests to all IP addresses and returns first response
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// copy request
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			errLog.Printf("Failed to read request body: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(r.Body)

		// forward request to all IP addresses
		ips, err := net.LookupIP(domain)
		if err != nil {
			errLog.Printf("Failed to resolve IP addresses for domain %s: %v", domain, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var responses []*http.Response

		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				addr := ipv4.String()

				req, err := http.NewRequest(r.Method, "http://"+addr+r.URL.Path, strings.NewReader(string(reqBody)))
				if err != nil {
					errLog.Printf("Failed to create request for IP %s: %v", ipv4, err)
					continue
				}

				for key, value := range r.Header {
					req.Header.Set(key, strings.Join(value, ","))
				}

				log.Printf("Forwarding request to %v\n", ipv4)
				resp, err := client.Do(req)
				if err != nil {
					errLog.Printf("Failed to forward request to IP %s: %v", ipv4, err)
					continue
				}

				responses = append(responses, resp)
			}
		}

		// return first response
		if len(responses) > 0 {
			resp := responses[0]
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {

				}
			}(resp.Body)

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				errLog.Printf("Failed to read response body: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			log.Println("Sending first response to the original request")
			_, _ = w.Write(respBody)
			return
		}

		// no successful responses
		errLog.Printf("No successful responses received")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	})

	// start http server
	serverAddr := net.JoinHostPort(httpServer, httpPort)
	log.Printf("Listening on %s", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, nil))
}
