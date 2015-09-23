package blob

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"uuid"
)

// The directory where blobs are stored
var StateDir = "/var/state"

type Blob struct {
	ID          uuid.UUID         `json:"id"`             // Blob ID
	Name        string            `json:"name"`           // original uploaded filename
	ContentType string            `json:"content_type"`   // MIME type of blob
	Size        int64             `json:"size"`           // Filesize in bytes
	Meta        map[string]string `json:"meta,omitempty"` // Freeform meta data detected about the file

}

// Get the timestamp that the blob was created
func (b *Blob) Time() time.Time {
	return b.ID.Time()
}

func (b *Blob) mkdir() error {
	dir, err := b.dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0777)
}

// Valid returns true if the blob has a valid ID.
func (b *Blob) Valid() bool {
	return b.ID.Valid()
}

// dir returns the pideon hole that the blob lives in /<StateDir>/YYYY/MM/DD/...
func (b *Blob) dir() (string, error) {
	if !b.Valid() {
		return "", errors.New("attempt to generate path for ")
	}
	if StateDir == "" {
		return "", errors.New("invalid state dir")
	}
	time := b.ID.Time()
	path := []string{
		StateDir,
		time.Format("2006"),
		time.Format("01"),
		time.Format("06"),
	}
	return filepath.Join(path...), nil
}

// path returns the path to the blob data
func (b *Blob) path() (string, error) {
	dir, err := b.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, b.ID.String()), nil
}

// Exists returns true if the blob is valid and has a data file
func (b *Blob) Exists() bool {
	if !b.Valid() {
		return false
	}
	filename, err := b.Path()
	if err != nil {
		return false
	}
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

// path returns the local filesystem path of to the blob data
// Blobs are stored in the state dir at /YYYY/MM/DD/UUID
// The UUID is always a v1, so this path can be calculated
// from the UUID alone.
func (b *Blob) Path() (string, error) {
	return b.path()
}

// metadataPath returns the local filesystem path of the metadata json
func (b *Blob) metadataPath() (string, error) {
	path, err := b.path()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.json", path), nil
}

func (b *Blob) unmarshal() error {
	path, err := b.metadataPath()
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	err = dec.Decode(b)
	if err != nil {
		return err
	}
	return nil
}

func (b *Blob) marshal() error {
	if err := b.mkdir(); err != nil {
		return err
	}
	path, err := b.metadataPath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	err = enc.Encode(b)
	if err != nil {
		return err
	}
	return nil
}

func (b *Blob) WriteFrom(src io.Reader) error {
	if err := b.mkdir(); err != nil {
		return err
	}
	path, err := b.path()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	b.Size, err = io.Copy(f, src)
	if err != nil {
		return err
	}
	if err := b.marshal(); err != nil {
		return nil
	}
	return nil
}

// File returns a new open read-only *os.File for the blob data.
// Users must close the file.
func (b *Blob) File() (*os.File, error) {
	if !b.Exists() {
		return nil, fmt.Errorf("blob does not have any data to read")
	}
	path, err := b.path()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// New returns a *Blob with a new ID set
func New() *Blob {
	b := &Blob{}
	b.ID = uuid.TimeUUID()
	return b
}

// Get loads the meta data and returns the *Blob for a given UUID
func Get(id uuid.UUID) (*Blob, error) {
	b := &Blob{
		ID: id,
	}
	if err := b.unmarshal(); err != nil {
		return nil, err
	}
	return b, nil
}
