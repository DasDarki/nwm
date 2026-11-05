package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/fsutil"
)

type ProgressFunc func(done int64, total int64)

func File(ctx context.Context, url string, destination string, onProgress ProgressFunc) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "nwm/"+buildinfo.Version)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: unexpected status %s", url, response.Status)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	partial := destination + ".part"
	file, err := os.Create(partial)
	if err != nil {
		return err
	}
	counter := &progressWriter{total: response.ContentLength, onProgress: onProgress}
	_, copyErr := io.Copy(io.MultiWriter(file, counter), response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(partial)
		return fmt.Errorf("downloading %s: %w", url, copyErr)
	}
	if closeErr != nil {
		os.Remove(partial)
		return closeErr
	}
	return os.Rename(partial, destination)
}

type progressWriter struct {
	done       int64
	total      int64
	onProgress ProgressFunc
}

func (w *progressWriter) Write(chunk []byte) (int, error) {
	w.done += int64(len(chunk))
	if w.onProgress != nil {
		w.onProgress(w.done, w.total)
	}
	return len(chunk), nil
}

func ExtractTarGz(archive string, destination string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	decompressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer decompressed.Close()
	reader := tar.NewReader(decompressed)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := fsutil.SafeJoin(destination, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(target, reader, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

func ExtractZip(archive string, destination string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, entry := range reader.File {
		target, err := fsutil.SafeJoin(destination, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		content, err := entry.Open()
		if err != nil {
			return err
		}
		err = writeFile(target, content, entry.Mode())
		content.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(target string, content io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if mode.Perm() == 0 {
		mode = 0o644
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, content); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
