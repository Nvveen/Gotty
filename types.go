package gotty

type TermInfo struct {
	boolAttributes map[string]bool
	numAttributes  map[string]int16
	strAttributes  map[string]string
	// The various names of the TermInfo file.
	Names []string
}
