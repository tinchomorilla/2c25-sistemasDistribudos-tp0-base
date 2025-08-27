package common

import (
    "bufio"
    "encoding/json"
    "fmt"
    "net"
)

// SendMessage serializes message to JSON and sends with newline
func SendMessage(conn net.Conn, message interface{}) error {
    data, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("error marshaling message: %w", err)
    }
    data = append(data, '\n') // delimiter
    _, err = conn.Write(data)
    return err
}

// RecvMessage reads until newline and decodes JSON
func RecvMessage(conn net.Conn, v interface{}) error {
    reader := bufio.NewReader(conn)
    line, err := reader.ReadBytes('\n')
    if err != nil {
        return fmt.Errorf("error reading message: %w", err)
    }
    return json.Unmarshal(line, v)
}
