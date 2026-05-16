package logger

import (
	"fmt"
	"os"
	"time"
)

const logFile = "logs/vpsctl.log"

func Log(server, command, result string) error {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s — %q\n%s\n---\n", timestamp, server, command, result)

	_, err = f.WriteString(entry)
	return err
}