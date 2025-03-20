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

func serveHTTP(rootDir string) *http.Server {
	var server *http.Server = &http.Server{Addr: lib.HTTP_PORT}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var fPath string = path.Join(rootDir, r.URL.Path)

		fmt.Println(fPath)

		if !strings.HasPrefix(fPath, rootDir) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		if _, err := os.Stat(fPath); os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, fPath)
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	return server
}

func Serve(rootDir string, serveHTTPFallbackDir string) (quit chan bool, err error) {
	quit = make(chan bool)

	var (
		addr *net.UDPAddr
		conn *net.UDPConn
	)

	if addr, err = net.ResolveUDPAddr("udp4", lib.PORT); err != nil {
		return nil, err
	}

	if conn, err = net.ListenUDP("udp4", addr); err != nil {
		return nil, err
	}

	var server *http.Server = nil

	if serveHTTPFallbackDir != "" {
		server = serveHTTP(serveHTTPFallbackDir)
	}

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

					var fPath string = path.Join(rootDir, filename)
					log.Warningf("Received RRQ request for %s from %s\n", fPath, clientAddr.String())

					if !strings.HasPrefix(fPath, rootDir) {
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
