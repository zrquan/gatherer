package input

import (
	"bufio"
	"os"
)

type Wordlist struct {
	data     [][]byte
	position int
}

func NewWordlist(path string) (*Wordlist, error) {
	var wl Wordlist
	wl.position = -1
	valid, err := wl.validFile(path)
	if err != nil {
		return &wl, err
	}
	if valid {
		err = wl.readFile(path)
	}
	return &wl, err
}

// Next will increment the cursor position, and return a boolean telling if there's words left in the list
func (w *Wordlist) Next() bool {
	w.position++
	return w.position < len(w.data)
}

// Value returns the value from wordlist at current cursor position
func (w *Wordlist) Value() []byte {
	return w.data[w.position]
}

// Total returns the size of wordlist
func (w *Wordlist) Total() int {
	return len(w.data)
}

// validFile checks that the wordlist file exists and can be read
func (w *Wordlist) validFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	f.Close()
	return true, nil
}

// readFile reads the file line by line to a byte slice
func (w *Wordlist) readFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var data [][]byte
	reader := bufio.NewScanner(file)
	for reader.Scan() {
		data = append(data, []byte(reader.Text()))
	}
	w.data = data
	return reader.Err()
}
