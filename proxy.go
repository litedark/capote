package main

import (
	"html"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const errstr = "<h1>This is a captive portal - onion sites only</h1>\n"

type interceptor struct {
	http.Handler
}

func (i *interceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.Host, ".onion") {
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		switch len(portal) {
		case 0:
			io.WriteString(w, errstr)
		default:
			f, err := os.Open(portal)
			if err != nil {
				io.WriteString(w, errstr)
				io.WriteString(w, "<pre>"+html.EscapeString(err.Error())+"</pre>\n")
				return
			}
			defer f.Close()
			io.Copy(w, f)
		}
		return
	}
	i.Handler.ServeHTTP(w, r)
}

func prox(errchan chan<- error) {
	l, err := net.Listen("tcp4", *httpaddr)
	if err != nil {
		errchan <- err
		return
	}
	socks, err := proxy.SOCKS5("tcp4", *socksaddr, nil, &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: false,
	})
	if err != nil {
		errchan <- err
		return
	}
	transport := &http.Transport{
		Dial:                  socks.Dial,
		MaxIdleConns:          50,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	}
	p := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = r.Host
			r.URL.Scheme = "http"
		},
		Transport:     transport,
		FlushInterval: 50 * time.Millisecond,
		ErrorLog:      log.New(ioutil.Discard, "", 0),
	}
	errchan <- http.Serve(l, &interceptor{p})
}
