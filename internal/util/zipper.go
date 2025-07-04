package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ZipSource struct {
	Path  string
	Alias string
}

func ZipDirectory(sourcePath, targetZipPath string) error {
	sources := []ZipSource{{
		Path:  sourcePath,
		Alias: filepath.Base(sourcePath),
	}}
	return ZipMultipleDirectories(sources, targetZipPath)
}

func ZipMultipleDirectories(sources []ZipSource, targetZipPath string) error {
	log.Printf("📦 Creating zip archive at %s", targetZipPath)

	zipfile, err := os.Create(targetZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	for _, source := range sources {
		if _, err := os.Stat(source.Path); os.IsNotExist(err) {
			log.Printf("⚠️ Source path not found, skipping: %s", source.Path)
			continue
		}

		err := filepath.Walk(source.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = filepath.Join(source.Alias, strings.TrimPrefix(path, source.Path))

			if info.IsDir() {
				header.Name += "/"
			} else {

				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = io.Copy(writer, file)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	log.Printf("✅ Zip archive created successfully.")
	return nil
}

func Unzip(sourceZipPath, destination string) error {
	log.Printf("📂 Unzipping %s to %s", sourceZipPath, destination)
	r, err := zip.OpenReader(sourceZipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(destination, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	log.Printf("✅ Unzip operation completed.")
	return nil
}

func UntarGz(sourceTarballPath, destination string) error {
	log.Printf("📂 Extracting tarball %s to %s", sourceTarballPath, destination)

	file, err := os.Open(sourceTarballPath)
	if err != nil {
		return fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		target := filepath.Join(destination, header.Name)

		if !strings.HasPrefix(target, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tarball: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory from tarball: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for file: %w", err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file from tarball: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file content from tarball: %w", err)
			}
			outFile.Close()
		}
	}
	log.Printf("✅ Tarball extraction completed.")
	return nil
}
