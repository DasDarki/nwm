package recipe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/pelletier/go-toml/v2"
)

type Format string

const (
	FormatTOML Format = "toml"
	FormatDSL  Format = "dsl"
)

func ParseFormat(text string) (Format, error) {
	switch strings.ToLower(text) {
	case "toml":
		return FormatTOML, nil
	case "dsl", "recipefile":
		return FormatDSL, nil
	default:
		return "", fmt.Errorf("unknown format %q, use toml or dsl", text)
	}
}

func (f Format) FileName() string {
	if f == FormatDSL {
		return DSLFileName
	}
	return FileName
}

type Document struct {
	Dir      string
	FileName string
	Format   Format
	Manifest *Manifest
}

func (d *Document) Path() string {
	if d.FileName != "" {
		return filepath.Join(d.Dir, d.FileName)
	}
	return filepath.Join(d.Dir, d.Format.FileName())
}

func IsRecipeFileName(name string) bool {
	lowered := strings.ToLower(name)
	return lowered == strings.ToLower(DSLFileName) || strings.HasSuffix(lowered, ".recipefile") || strings.HasSuffix(lowered, ".recipe") || strings.HasSuffix(lowered, ".toml")
}

func FormatForFile(name string) Format {
	if strings.HasSuffix(strings.ToLower(name), ".toml") {
		return FormatTOML
	}
	return FormatDSL
}

func OpenDocumentFile(path string) (*Document, error) {
	document := &Document{Dir: filepath.Dir(path), FileName: filepath.Base(path), Format: FormatForFile(path)}
	if err := document.load(); err != nil {
		return nil, err
	}
	return document, nil
}

func NewDocumentFile(path string, name string) *Document {
	document := NewDocument(filepath.Dir(path), FormatForFile(path), name)
	document.FileName = filepath.Base(path)
	return document
}

func (d *Document) load() error {
	var err error
	if d.Format == FormatDSL {
		content, readErr := os.ReadFile(d.Path())
		if readErr != nil {
			return readErr
		}
		d.Manifest, err = ParseDSL(string(content))
		if err != nil {
			return fmt.Errorf("%s: %w", d.Path(), err)
		}
		return nil
	}
	d.Manifest, err = loadTOML(d.Path())
	return err
}

func convertedFileName(name string, format Format) string {
	if name == "" || name == DSLFileName || name == FileName {
		return ""
	}
	base := name
	for _, suffix := range []string{".toml", ".Recipefile", ".recipefile", ".recipe"} {
		base = strings.TrimSuffix(base, suffix)
	}
	if format == FormatTOML {
		return base + ".toml"
	}
	return base + ".Recipefile"
}

func Detect(directory string) (Format, bool, error) {
	hasTOML := fsutil.Exists(filepath.Join(directory, FileName))
	hasDSL := fsutil.Exists(filepath.Join(directory, DSLFileName))
	switch {
	case hasTOML && hasDSL:
		return "", false, fmt.Errorf("%s contains both %s and %s, keep only one", directory, FileName, DSLFileName)
	case hasDSL:
		return FormatDSL, true, nil
	case hasTOML:
		return FormatTOML, true, nil
	default:
		return "", false, nil
	}
}

func OpenDocument(directory string) (*Document, error) {
	format, found, err := Detect(directory)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no %s or %s in %s", DSLFileName, FileName, directory)
	}
	document := &Document{Dir: directory, Format: format}
	if err := document.load(); err != nil {
		return nil, err
	}
	return document, nil
}

func NewDocument(directory string, format Format, name string) *Document {
	return &Document{Dir: directory, Format: format, Manifest: &Manifest{Recipe: Meta{Name: name, Version: "0.1.0"}}}
}

func Render(manifest *Manifest, format Format) (string, error) {
	if format == FormatDSL {
		return RenderDSL(manifest), nil
	}
	content, err := toml.Marshal(manifest)
	return string(content), err
}

func (d *Document) Render() (string, error) {
	return Render(d.Manifest, d.Format)
}

func (d *Document) Save() error {
	content, err := d.Render()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(d.Path(), []byte(content), 0o644)
}

func (d *Document) Convert(format Format) error {
	if format == d.Format {
		return nil
	}
	previous := d.Path()
	d.FileName = convertedFileName(d.FileName, format)
	d.Format = format
	if err := d.Save(); err != nil {
		return err
	}
	if err := os.Remove(previous); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
