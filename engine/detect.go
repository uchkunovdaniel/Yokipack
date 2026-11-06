package engine

import (
	"fmt"
)

func DetectLookupTable() {
	fmt.Println(isLanguageKnown())
}

func (fs *OSFileSystem) findContainerfiles() (string, bool) {
	containerFiles := []string{"Dockerfile", "Containerfile"}
	for _, name := range containerFiles {
		if fs.exists(name) {
			return fs.resolve(name), true
		}
	}
	return "", false
}
