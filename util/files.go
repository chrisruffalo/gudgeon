package util

import (
	"bufio"
	"io"
	"os"
)

// from: https://stackoverflow.com/questions/24562942/golang-how-do-i-determine-the-number-of-lines-in-a-file-efficiently
func LinesInFile(path string) (uint, error) {

	// open file
	r, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	// create buffered reader (more raw than scanner/readline)
    reader := bufio.NewReader(r)

    // start with 0
    count := uint(0)

    // chomps through the file/reader in buffer size chunks
    for {
        _, isPrefix, err := reader.ReadLine()

        if !isPrefix {
            count++
        }

        if err == io.EOF {
            return count - 1, nil
        } else if err != nil {
            return count, err
        }

    }

    return count, nil
}