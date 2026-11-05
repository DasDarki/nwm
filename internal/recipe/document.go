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
	Format   Format
	Manifest *Manifest
}

func (d *Document) Path() string {
	return filepath.Join(d.Dir, d.Format.FileName())
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
	switch format {
	case FormatDSL:
		content, err := os.ReadFile(document.Path())
		if err != nil {
			return nil, err
		}
		if document.Manifest, err = ParseDSL(string(content)); err != nil {
			return nil, err
		}
	default:
		if document.Manifest, err = loadTOML(document.Path()); err != nil {
			return nil, err
		}
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
	d.Format = format
	if err := d.Save(); err != nil {
		return err
	}
	if err := os.Remove(previous); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
