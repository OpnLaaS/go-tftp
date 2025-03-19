package gotftp

import (
	"net"

	"github.com/OpnLaaS/go-tftp/lib"
	"github.com/z46-dev/go-logger"
)

func Serve() (quit chan bool, err error) {
	quit = make(chan bool)

	var (
		addr *net.UDPAddr
		conn *net.UDPConn
	)

	if addr, err = net.ResolveUDPAddr("udp", lib.PORT); err != nil {
		return nil, err
	}

	if conn, err = net.ListenUDP("udp", addr); err != nil {
		return nil, err
	}

	go func() {
		defer conn.Close()

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

				log.Basicf("Received %d bytes from %s\n", bytesRead, clientAddr.String())

				opcode = int(buffer[1])

				if opcode == lib.OPCODE_RRQ {
					if filename, _, err = lib.ParseRQQRequest(buffer[:bytesRead]); err != nil {
						log.Error(err.Error())
						continue
					}

					log.Warningf("Received RRQ request for %s from %s\n", filename, clientAddr.String())
					if err = lib.SendFile(conn, clientAddr, filename); err != nil {
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
