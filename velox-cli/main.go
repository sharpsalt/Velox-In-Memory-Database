// filepath: velox-cli/main.go
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Connect to the Velox server
	conn, err := net.Dial("tcp", "localhost:7379")
	if err != nil {
		fmt.Println("Could not connect to Velox:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to Velox Database!")
	fmt.Println("Type your commands below (e.g., SET name srijan, GET name). Type 'exit' to quit.")

	reader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(conn)

	for {
		fmt.Print("velox> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			break
		}
		if input == "" {
			continue
		}

		// Split input into arguments
		args := strings.Split(input, " ")
		
		// Convert to RESP Array format
		// E.g., SET name srijan -> *3\r\n$3\r\nSET\r\n$4\r\nname\r\n$6\r\nsrijan\r\n
		respCmd := fmt.Sprintf("*%d\r\n", len(args))
		for _, arg := range args {
			respCmd += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
		}

		// Send to server
		_, err := conn.Write([]byte(respCmd))
		if err != nil {
			fmt.Println("Error sending command:", err)
			continue
		}

		// Read response from server (simplified for basic replies)
		// Note: A robust CLI would use a proper RESP parser here
		response := make([]byte, 1024)
		n, err := serverReader.Read(response)
		if err != nil {
			fmt.Println("Error reading response:", err)
			continue
		}

		// Print raw server response for now
		fmt.Print(string(response[:n]))
	}
}
