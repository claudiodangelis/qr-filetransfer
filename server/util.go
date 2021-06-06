package server

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func serveTemplate(name string, tmpl string, w io.Writer, data interface{}) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		panic(err)
	}
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

// getFileName generates a file name based on the existing files in the directory
// if name isn't taken leave it unchanged
// else change name to format "name(number).ext"
func getFileName(newFilename string, fileNamesInTargetDir []string) string {
	fileExt := filepath.Ext(newFilename)
	fileName := strings.TrimSuffix(newFilename, fileExt)
	number := 1
	i := 0
	for i < len(fileNamesInTargetDir) {
		if newFilename == fileNamesInTargetDir[i] {
			newFilename = fmt.Sprintf("%s(%v)%s", fileName, number, fileExt)
			number++
			i = 0
		}
		i++
	}
	return newFilename
}

func sanitizeFileName(fileName string) string {
	replStr := "_"
	switch runtime.GOOS {
	case "windows":
		// Replace the characters reserved by Windows
		// Reference: https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
		re := regexp.MustCompile(`[<>:"/\\|?*]`)
		return re.ReplaceAllLiteralString(fileName, replStr)
	default:
		// Replace the path separator character
		return strings.Replace(fileName, string(filepath.Separator), replStr, -1)
	}
}
