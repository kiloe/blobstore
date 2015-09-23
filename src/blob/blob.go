package blob

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
	"uuid"
)

// The directory where blobs are stored
var StateDir = "/var/state/blobstore"

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
	return os.MkdirAll(b.dir(), 0777)
}

// Valid returns true if the blob has a valid ID.
func (b *Blob) Valid() bool {
	return b.ID.Valid()
}

// dir returns the local filesystem path of to the blob dir
// Blobs are stored in the state dir at /YYYY/MM/DD/UUID
// The UUID is always a v1, so this path can be calculated
// from the UUID alone.
// Since calling this method implies something is about to
// be read/written it will panic if used on an invalid blob.
func (b *Blob) dir() string {
	if !b.Valid() {
		panic("attempt to generate path for ")
	}
	if StateDir == "" {
		panic("invalid state dir")
	}
	time := b.ID.Time()
	path := []string{
		StateDir,
		time.Format("2006"),
		time.Format("01"),
		time.Format("06"),
		b.ID.String(),
	}
	return filepath.Join(path...)
}

// Dir is the non-panicy version of dir
func (b *Blob) Dir() (string, error) {
	if StateDir == "" {
		return "", errors.New("invalid state dir")
	}
	if !b.Valid() {
		return "", errors.New("invalid blob")
	}
	return b.dir(), nil
}

// Path returns the local filesystem path of the actual blob data
func (b *Blob) Path() string {
	return filepath.Join(b.dir(), "data")
}

// metadataPath returns the local filesystem path of the metadata json
func (b *Blob) metadataPath() string {
	return filepath.Join(b.dir(), "meta.json")
}

func (b *Blob) unmarshal() error {
	f, err := os.Open(b.metadataPath())
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
	f, err := os.OpenFile(b.metadataPath(), os.O_WRONLY|os.O_CREATE, 0666)
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
	f, err := os.OpenFile(b.Path(), os.O_WRONLY|os.O_CREATE, 0666)
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

func (b *Blob) NewReader() (io.ReadCloser, error) {
	f, err := os.OpenFile(b.Path(), os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func New() *Blob {
	b := &Blob{}
	b.ID = uuid.TimeUUID()
	return b
}

func Get(id uuid.UUID) (*Blob, error) {
	b := &Blob{
		ID: id,
	}
	if err := b.unmarshal(); err != nil {
		return nil, err
	}
	return b, nil
}
