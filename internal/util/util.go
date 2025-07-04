package util

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func DownloadFile(url, targetPath string) error {
	log.Printf("⬇️ Downloading from %s to %s", url, targetPath)

	if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory for download: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http.Get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("os.Create failed: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("io.Copy failed: %w", err)
	}

	return nil
}

func SanitizeFileName(name string) string {

	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := re.ReplaceAllString(name, "")

	return strings.Trim(sanitized, " .")
}
