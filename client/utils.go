package client

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func readLineFromFile(file *os.File) ([]string, error) {
	var line []string
	for {
		var oneWord string
		n, err := fmt.Fscan(file, &oneWord)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if n != 0 {
			line = append(line, oneWord)
			// Item ends with }
			if strings.Contains(oneWord, "}") {
				// // if
				// if len(line) == 1 {
				// 	line = []string{}
				// 	continue
				// }
				return line, nil
			}
		} else {
			time.Sleep(time.Second)
		}
	}
}

func setInterval(function func(), intervalSec time.Duration) *time.Ticker {
	ticker := time.NewTicker(intervalSec * time.Second)
	go func() {
		for {
			<-ticker.C
			function()
		}
	}()
	return ticker
}

func createDeduplicationId(data string) string {
	id := fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString([]byte(data)))
	if len(id) > 128 {
		return id[:128]
	}
	return id
}
