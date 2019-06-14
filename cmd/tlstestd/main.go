// Command tlstestd loads the given TLS cert/key and listens to the given port over HTTPS.
package main

import (
	"flag"
	"log"
	"net/http"

	"within.website/x/internal"
)

var (
	cert = flag.String("cert", "cert.pem", "TLS cert file")
	key  = flag.String("key", "key.pem", "TLS key")
	port = flag.String("port", "2848", "TCP port to listen on")
)

func helloServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Your TLS connection works...or you accepted an invalid cert :)"))
}

func main() {
	internal.HandleStartup()

	http.HandleFunc("/", helloServer)
	err := http.ListenAndServeTLS(":"+*port, *cert, *key, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
