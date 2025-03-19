package gotftp

import (
	"net"

	"github.com/OpnLaaS/go-tftp/lib"
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

	defer conn.Close()

	go func() {
		var (
			bytesRead  int    = 0
			buffer     []byte = make([]byte, 1024)
			clientAddr *net.UDPAddr
			filename   string
			opcode     int
			err        error
		)

		for {
			select {
			case <-quit:
				return
			default:
				if bytesRead, clientAddr, err = conn.ReadFromUDP(buffer); err != nil || bytesRead < 4 {
					continue
				}

				opcode = int(buffer[1])

				if opcode == lib.OPCODE_RRQ {
					if filename, _, err = lib.ParseRQQRequest(buffer[:bytesRead]); err != nil {
						continue
					}

					lib.SendFile(conn, clientAddr, filename)
				}
			}
		}

	}()

	return quit, nil
}
