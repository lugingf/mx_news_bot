package pdfconverter

import (
	"fmt"
	"os"
	"os/exec"
)

func ConvertPDFToText(pdfFile, textFile string) error {
	fmt.Printf("Converting PDF %s to text\n", pdfFile)
	cmd := exec.Command("pdftotext", "-f", "1", "-l", "1", "-layout", pdfFile, textFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
