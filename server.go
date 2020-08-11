package fins

import (
	"encoding/binary"
	"log"
	"net"
)

// Server Omron FINS server
type Server struct {
	addr      Address
	conn      *net.UDPConn
	dmarea    []byte
	bitdmarea []byte
	closed    bool
}

// Address A full device address
type Address struct {
	finsAddress
	udpAddress *net.UDPAddr
}
type finsAddress struct {
	network byte
	node    byte
	unit    byte
}

//NewServer Create new server
func NewServer(IP string, port int, network byte, node byte, unit byte) (*Server, error) {

	s := new(Server)
	s.addr = createAddress(IP, port, network, node, unit)
	s.dmarea = make([]byte, DMAreaSize)
	s.bitdmarea = make([]byte, DMAreaSize)

	conn, err := net.ListenUDP("udp", s.addr.udpAddress)
	if err != nil {
		return nil, err
	}
	s.conn = conn
	return s, nil
}

//Listen Listen the FINS server
func (s *Server) Listen() {
	var buf [1024]byte
	for {
		rlen, remote, err := s.conn.ReadFromUDP(buf[:])
		if rlen > 0 {
			req := decodeRequest(buf[:rlen])
			resp := s.handler(req)

			_, err = s.conn.WriteToUDP(encodeResponse(resp), &net.UDPAddr{IP: remote.IP, Port: remote.Port})
		}
		if err != nil {
			// do not complain when connection is closed by user
			if !s.closed {
				log.Fatal("Encountered error in server loop: ", err)
			}
			break
		}
	}
}

// Works with only DM area, 2 byte integers
func (s *Server) handler(r request) response {
	var endCode uint16
	data := []byte{}
	switch r.commandCode {
	case CommandCodeMemoryAreaRead, CommandCodeMemoryAreaWrite:
		memAddr := decodeMemoryAddress(r.data[:4])
		ic := binary.BigEndian.Uint16(r.data[4:6]) // Item count

		switch memAddr.memoryArea {
		case MemoryAreaDMWord:

			if memAddr.address+ic*2 > DMAreaSize { // Check address boundary
				endCode = EndCodeAddressRangeExceeded
				break
			}

			if r.commandCode == CommandCodeMemoryAreaRead { //Read command
				data = s.dmarea[memAddr.address : memAddr.address+ic*2]
			} else { // Write command
				copy(s.dmarea[memAddr.address:memAddr.address+ic*2], r.data[6:6+ic*2])
			}
			endCode = EndCodeNormalCompletion

		case MemoryAreaDMBit:
			if memAddr.address+ic > DMAreaSize { // Check address boundary
				endCode = EndCodeAddressRangeExceeded
				break
			}
			start := memAddr.address + uint16(memAddr.bitOffset)
			if r.commandCode == CommandCodeMemoryAreaRead { //Read command
				data = s.bitdmarea[start : start+ic]
			} else { // Write command
				copy(s.bitdmarea[start:start+ic], r.data[6:6+ic])
			}
			endCode = EndCodeNormalCompletion

		default:
			log.Printf("Memory area is not supported: 0x%04x\n", memAddr.memoryArea)
			endCode = EndCodeNotSupportedByModelVersion
		}

	default:
		log.Printf("Command code is not supported: 0x%04x\n", r.commandCode)
		endCode = EndCodeNotSupportedByModelVersion
	}
	return response{defaultResponseHeader(r.header), r.commandCode, endCode, data}
}

// Close Closes the FINS server
func (s *Server) Close() {
	s.closed = true
	s.conn.Close()
}
func createAddress(ip string, port int, network, node, unit byte) Address {
	return Address{
		udpAddress: &net.UDPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		},
		finsAddress: finsAddress{
			network: network,
			node:    node,
			unit:    unit,
		},
	}
}
