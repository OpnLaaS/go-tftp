package gotftp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/OpnLaaS/go-tftp/lib"
	"github.com/z46-dev/go-logger"
)

func serveHTTP(rootDir, httpAddr string) *http.Server {
	var mux *http.ServeMux = http.NewServeMux()
	var log *logger.Logger = logger.NewLogger().SetPrefix("[HTTP]", logger.BoldGreen).IncludeTimestamp()

	log.Statusf("Server started on %s\n", httpAddr)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var fPath string = path.Join(rootDir, r.URL.Path)
		log.Warningf("Serving %s\n", fPath)

		if !strings.HasPrefix(fPath, rootDir) {
			http.Error(w, "File not found", http.StatusNotFound)
			log.Errorf("File not found: %s\n", fPath)
			return
		}

		if _, err := os.Stat(fPath); os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			log.Errorf("File not found: %s\n", fPath)
			return
		}

		http.ServeFile(w, r, fPath)
		log.Successf("Served %s\n", fPath)
	})

	var server *http.Server = &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Error(err.Error())
		}

		log.Status("Server stopped")
	}()

	return server
}

type TFTPOptions struct {
	RootDir   string
	TFTP_Address string

	ServeHttp    bool
	HTTP_RootDir string
	HTTP_Address string
}

func Serve(options TFTPOptions) (quit chan bool, err error) {
	if options.RootDir == "" {
		return nil, fmt.Errorf("config err: options.RootDir is required")
	}

	if options.TFTP_Address == "" {
		return nil, fmt.Errorf("config err: options.TFTP_Address is required")
	}

	var (
		addr *net.UDPAddr
		conn *net.UDPConn
	)

	if addr, err = net.ResolveUDPAddr("udp4", options.TFTP_Address); err != nil {
		return nil, err
	}

	if conn, err = net.ListenUDP("udp4", addr); err != nil {
		return nil, err
	}

	var server *http.Server = nil

	if options.ServeHttp {
		if options.HTTP_RootDir == "" {
			return nil, fmt.Errorf("config err: options.HTTP_RootDir is required when ServeHttp is true")
		}

		if options.HTTP_Address == "" {
			return nil, fmt.Errorf("config err: options.HTTP_Address is required when ServeHttp is true")
		}

		server = serveHTTP(options.HTTP_RootDir, options.HTTP_Address)
	}

	quit = make(chan bool)

	go func() {
		defer conn.Close()

		if server != nil {
			defer server.Shutdown(context.TODO())
		}

		var (
			bytesRead  int    = 0
			buffer     []byte = make([]byte, 1024)
			clientAddr *net.UDPAddr
			filename   string
			opcode     int
			err        error
			log        *logger.Logger = logger.NewLogger().SetPrefix("[TFTP]", logger.BoldPurple).IncludeTimestamp()
		)

		log.Status("Server started")

		for {
			select {
			case <-quit:
				log.Status("Server stopped due to quit signal")
				return
			default:
				if bytesRead, clientAddr, err = conn.ReadFromUDP(buffer); err != nil {
					log.Error(err.Error())
					continue
				}

				if bytesRead < 4 {
					log.Errorf("Invalid request from %s\n", clientAddr.String())
					continue
				}

				opcode = int(buffer[1])

				if opcode == lib.OPCODE_RRQ {
					if filename, _, err = lib.ParseRQQRequest(buffer[:bytesRead]); err != nil {
						log.Errorf("Failed to parse RRQ request for %s: %s\n", clientAddr.String(), err.Error())
						continue
					}

					var fPath string = path.Join(options.RootDir, filename)
					log.Warningf("Received RRQ request for %s from %s\n", fPath, clientAddr.String())

					if !strings.HasPrefix(fPath, options.RootDir) {
						lib.SendError(conn, addr, 1, "File not found")
					} else if err = lib.SendFile(conn, clientAddr, fPath); err != nil {
						log.Errorf("Failed to send file: %s\n", err.Error())
					} else {
						log.Successf("File %s sent to %s\n", filename, clientAddr.String())
					}
				}
			}
		}

	}()

	return quit, nil
}
