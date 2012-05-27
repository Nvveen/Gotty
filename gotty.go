// Gotty is a Go-package for reading and parsing the terminfo database
package gotty

// TODO fix full-name attribute lookup

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Open a terminfo file by the name given and construct a TermInfo object.
// If something went wrong reading the terminfo database file, an error is
// returned.
func OpenTermInfo(termName string) (*TermInfo, error) {
	var term *TermInfo
	var err error
	// Find the environment variables
	termloc := os.Getenv("TERMINFO")
	if len(termloc) == 0 {
		// Search like ncurses
		locations := []string{os.Getenv("HOME") + "/.terminfo/", "/etc/terminfo/",
			"/lib/terminfo/", "/usr/share/terminfo/"}
		var path string
		for _, str := range locations {
			// Construct path
			path = str + string(termName[0]) + "/" + termName
			// Check if path can be opened
			file, _ := os.Open(path)
			if file != nil {
				// Path can open, fall out and use current path
				file.Close()
				break
			}
		}
		if len(path) > 0 {
			term, err = readTermInfo(path)
		} else {
			err = errors.New(fmt.Sprintf("No terminfo file(-location) found"))
		}
	}
	return term, err
}

// Open a terminfo file from the environment variable containing the current
// terminal name and construct a TermInfo object. If something went wrong
// reading the terminfo database file, an error is returned.
func OpenTermInfoEnv() (*TermInfo, error) {
	termenv := os.Getenv("TERM")
	return OpenTermInfo(termenv)
}

// Return an attribute by the name attr provided. If none can be found,
// an error is returned.
func (term *TermInfo) GetAttribute(attr string) (stacker, error) {
	var value interface{}
	var ok bool
	// Look inside bool
	value, ok = term.boolAttributes[attr]
	if ok {
		return value, nil
	}
	// Look inside numbers
	value, ok = term.numAttributes[attr]
	if ok {
		return value, nil
	}
	// Look inside strings
	value, ok = term.strAttributes[attr]
	if ok {
		return value, nil
	}
	// Doesn't exist, error
	return nil, errors.New(fmt.Sprintf("Could not find attribute"))
}

// This function takes a path to a terminfo file and reads it in binary
// form to construct the actual TermInfo file.
func readTermInfo(path string) (*TermInfo, error) {
	// Open the terminfo file
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	// magic, nameSize, boolSize, nrSNum, nrOffsetsStr, strSize
	// Header is composed of the magic 0432 octal number, size of the name
	// section, size of the boolean section, the amount of number values,
	// the number of offsets of strings, and the size of the string section.
	var header [6]int16
	// Byte array is used to read in byte values
	var byteArray []byte
	// Short array is used to read in short values
	var shArray []int16
	// TermInfo object to store values
	var term TermInfo

	// Read in the header
	err = binary.Read(file, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	// If magic number isn't there or isn't correct, we have the wrong filetype
	if header[0] != 0432 {
		return nil, errors.New(fmt.Sprintf("Wrong filetype"))
	}

	// Read in the names
	byteArray = make([]byte, header[1])
	err = binary.Read(file, binary.LittleEndian, &byteArray)
	if err != nil {
		return nil, err
	}
	term.Names = strings.Split(string(byteArray), "|")

	// Read in the booleans
	byteArray = make([]byte, header[2])
	err = binary.Read(file, binary.LittleEndian, &byteArray)
	if err != nil {
		return nil, err
	}
	term.boolAttributes = make(map[string]bool)
	for i, b := range byteArray {
		if b == 1 {
      term.boolAttributes[boolAttr[i*2+1]] = true
		}
	}
	// If the number of bytes read is not even, a byte for alignment is added
	if len(byteArray)%2 != 0 {
		err = binary.Read(file, binary.LittleEndian, make([]byte, 1))
		if err != nil {
			return nil, err
		}
	}

	// Read in shorts
	shArray = make([]int16, header[3])
	err = binary.Read(file, binary.LittleEndian, &shArray)
	if err != nil {
		return nil, err
	}
	term.numAttributes = make(map[string]int16)
	for i, n := range shArray {
		if n != 0377 && n > -1 {
			term.numAttributes[numAttr[i*2+1]] = n
		}
	}

	// Read the offsets into the short array
	shArray = make([]int16, header[4])
	err = binary.Read(file, binary.LittleEndian, &shArray)
	if err != nil {
		return nil, err
	}
	// Read the actual strings in the byte array
	byteArray = make([]byte, header[5])
	err = binary.Read(file, binary.LittleEndian, &byteArray)
	if err != nil {
		return nil, err
	}
	term.strAttributes = make(map[string]string)
	// We get an offset, and then iterate until the string is null-terminated
	for i, offset := range shArray {
		if offset > -1 {
			r := offset
			for ; byteArray[r] != 0; r++ {
			}
			term.strAttributes[strAttr[i*2+1]] = string(byteArray[offset:r])
		}
	}
	return &term, nil
}
