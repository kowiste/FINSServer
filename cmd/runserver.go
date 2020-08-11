package main

import fins "github.com/kowiste/FINSServer"

func main() {

	server, err := fins.NewServer("0.0.0.0", 9600, 0, 35, 0)
	if err != nil {
		panic(err)
	}
	server.Listen()

}
