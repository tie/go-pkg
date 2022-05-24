package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/alta/insecure"
	"go.uber.org/goleak"
	"go.uber.org/zap"

	"go.pact.im/x/httpserver"
	"go.pact.im/x/zaplog"
)

var (
	tcpAddress         = flag.String("tcp", ":80", "TCP address to listen on")
	udpAddress         = flag.String("udp", ":443", "UDP/QUIC address to listen on")
	tlsAddress         = flag.String("tls", ":443", "TLS address to listen on")
	tlsOptionalAddress = flag.String("tls-optional", ":8443", "OptionalTLS address to listen on")
	certFilePath       = flag.String("cert", "", "TLS cert file path")
	keyFilePath        = flag.String("key", "", "TLS cert file path")
	servePath          = flag.String("dir", ".", "directory to serve")
	enablePanic        = flag.Bool("panic", false, "enable /panic endpoint")
	enableAbort        = flag.Bool("abort", false, "enable /abort endpoint")
)

func main() {
	flag.Parse()
	os.Exit(main1())
}

var (
	rootLogger = zaplog.New(os.Stderr)
	logger     = rootLogger.Named("app")
)

func main1() int {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	defer signal.Stop(sigc)

	handler := http.FileServer(http.Dir(*servePath))
	if *enablePanic || *enableAbort {
		mux := http.NewServeMux()
		mux.Handle("/", handler)
		if *enablePanic {
			mux.HandleFunc("/panic", func(http.ResponseWriter, *http.Request) {
				panic("oopsie")
			})
		}
		if *enableAbort {
			mux.HandleFunc("/panic", func(http.ResponseWriter, *http.Request) {
				panic(http.ErrAbortHandler)
			})
		}
		handler = mux
	}

	var tlsConf *tls.Config
	if *udpAddress != "" || *tlsAddress != "" || *tlsOptionalAddress != "" {
		tlsConf = &tls.Config{
			Certificates: make([]tls.Certificate, 1),
		}

		var err error
		if *certFilePath != "" || *keyFilePath != "" {
			tlsConf.Certificates[0], err = tls.LoadX509KeyPair(*certFilePath, *keyFilePath)
		} else {
			tlsConf.Certificates[0], err = insecure.Cert()
		}
		if err != nil {
			logger.Error("Failed to load certificate", zap.Error(err))
			return 1
		}
	}

	var tcp []httpserver.StreamSocket
	var udp []httpserver.PacketSocket
	if addr := *tcpAddress; addr != "" {
		tcp = append(tcp, httpserver.TCP(addr))
	}
	if addr := *udpAddress; addr != "" {
		udp = append(udp, httpserver.UDP(addr))
	}
	if addr := *tlsAddress; addr != "" {
		tcp = append(tcp, httpserver.TLS(addr, tlsConf))
	}
	if addr := *tlsOptionalAddress; addr != "" {
		tcp = append(tcp, httpserver.OptionalTLS(addr, tlsConf, 0))
	}

	srv := httpserver.NewServer(httpserver.Options{
		Logger:  rootLogger.Named("http"),
		Handler: handler,
		H3: &httpserver.H3{
			TLSConfig: tlsConf,
		},
		StreamSockets: tcp,
		PacketSockets: udp,
	})
	err := srv.Run(context.Background(), func(ctx context.Context) error {
		logger.Info("Server has successfully started")
		select {
		case <-ctx.Done():
			logger.Info("Received server callback cancellation")
		case <-sigc:
			logger.Info("Received os.Interrupt signal")
		}
		logger.Info("Starting server shutdown")
		return nil
	})
	logger.Info("Server has completed shutdown")

	if err := goleak.Find(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if err != nil {
		logger.Error("Got error running the server", zap.Error(err))
		return 1
	}

	return 0
}
