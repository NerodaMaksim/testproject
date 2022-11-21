package client

import (
	"bufio"
	"io"
	"os"
	"time"
)

func SubscribeToFileInput(file *os.File) (chan string, chan error) {
	reader := bufio.NewReader(file)
	lines := make(chan string)
	errChan := make(chan error)
	go func() {
		for {
			line := make([]byte, 0)
			endOfLine := false
			for !endOfLine {
				partOfLine, isPrefix, err := reader.ReadLine()
				if err != nil && err != io.EOF {
					close(lines)
					errChan <- err
					break
				}

				endOfLine = !isPrefix

				if len(partOfLine) != 0 {
					line = append(line, partOfLine...)
				}

				if err == io.EOF {
					time.Sleep(time.Second)
				}
			}

			if len(line) != 0 {
				lines <- string(line)
			}
		}
	}()

	return lines, errChan
}
