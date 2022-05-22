package fins

//SetDMArea set the DM area
func (s *Server) SetDMArea(data []uint16) {
	s.dmarea = arrUint16ToByte(data)
}
