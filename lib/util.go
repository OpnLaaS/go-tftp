package lib

import (
	"fmt"
	"net"
	"os"
)

func SendError(conn *net.UDPConn, addr *net.UDPAddr, errCode int, errMsg string) (err error) {
	var buffer []byte = make([]byte, 5+len(errMsg))

	buffer[0] = 0
	buffer[1] = OPCODE_ERROR
	buffer[2] = 0
	buffer[3] = byte(errCode)
	copy(buffer[4:], errMsg)
	buffer[4+len(errMsg)] = 0

	_, err = conn.WriteToUDP(buffer, addr)
	return err
}

func ParseRQQRequest(buffer []byte) (file string, mode string, err error) {
	var (
		start int      = 2
		parts []string = make([]string, 0)
	)

	for i := 2; i < len(buffer); i++ {
		if buffer[i] == 0 {
			parts = append(parts, string(buffer[start:i]))
			start = i + 1
		}
	}

	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid request")
	}

	file = parts[0]
	mode = parts[1]
	return file, mode, nil
}

func SendFile(conn *net.UDPConn, addr *net.UDPAddr, filename string) (err error) {
    var file *os.File

    if file, err = os.Open(filename); err != nil {
        SendError(conn, addr, 1, "File not found")
        return err
    }
    defer file.Close()

    var (
        bytesRead int    = 0
        blockNum  uint16 = 1
        buffer    []byte = make([]byte, BLOCK_SIZE)
    )

    for {
        if bytesRead, err = file.Read(buffer); err != nil {
            return err
        }

        var dataPacket []byte = make([]byte, 4+bytesRead)
        dataPacket[0] = 0
        dataPacket[1] = OPCODE_DATA
        dataPacket[2] = byte(blockNum >> 8)
        dataPacket[3] = byte(blockNum)
        copy(dataPacket[4:], buffer[:bytesRead])

        if _, err = conn.WriteToUDP(dataPacket, addr); err != nil {
            return err
        }

	fmt.Println("Sending data chunk", blockNum) 

        var ack []byte = make([]byte, 4)
        if _, _, err = conn.ReadFromUDP(ack); err != nil {
            return err
        }

        if ack[1] != OPCODE_ACK || ack[2] != byte(blockNum>>8) || ack[3] != byte(blockNum) {
            return fmt.Errorf("invalid ACK received: %v", ack)
        }

        blockNum++
        if bytesRead < BLOCK_SIZE {
            return nil
        }
    }
}

