package main

import (
	"crypto/tls"

	"C"
	"fmt"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var dialTimeout = time.Duration(15) * time.Second
var path, _ = os.Getwd()

func httpClient(hello string, urlin string) (*http.Client, error){
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	parsedUrl, _ := url.Parse(urlin)

	var helloId utls.ClientHelloID

	switch hello {
	case "chrome":
		helloId = utls.HelloChrome_83
	case "chrome89":
		helloId = utls.HelloChrome_89
	case "ios":
		helloId = utls.HelloIOS_12_1
	case "firefox":
		helloId = utls.HelloFirefox_65
	default:
		helloId = utls.HelloGolang
	}

	//caCert, err := ioutil.ReadFile(path + "/server.crt")
	//if err != nil {
	//	log.Fatalf("Reading server certificate: %s", err)
	//}
	//caCertPool := x509.NewCertPool()
	//caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		//RootCAs: caCertPool,
		InsecureSkipVerify: true,
	}

	switch parsedUrl.Scheme{
	case "http":
		client.Transport = &http.Transport{TLSClientConfig: tlsConfig,
		}
	case "https":
		client.Transport = &http2.Transport{TLSClientConfig: tlsConfig, DialTLS: func(n, a string, cfg *tls.Config) (net.Conn, error) {
			config := utls.Config{ServerName: parsedUrl.Host, InsecureSkipVerify: true}
			dialConn, err := net.DialTimeout("tcp", parsedUrl.Host + ":443", dialTimeout)
			if err != nil {
				return nil, fmt.Errorf("net.DialTimeout error: %+v", err)
			}
			//uTLSConn := utls.UClient(dialConn, &config, utls.HelloCustom)
			uTLSConn := utls.UClient(dialConn, &config, helloId)

			err = uTLSConn.Handshake()
			if err != nil {
				return nil, fmt.Errorf("uTLSConn.Handshake() error: %+v", err)
			}

			return uTLSConn, nil
		},
		}

	default:
		log.Print("cant process scheme")
	}

	return &client, nil
}
