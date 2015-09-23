package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kingpin"
)

var (
	cli        = kingpin.New("blobstore", "file storage api")
	version    = cli.Version("0.0.1")
	verbose    = cli.Flag("verbose", "Verbose mode.").Short('v').Bool()
	secretKey  = cli.Flag("secret", "Secret key used during authentication").Default("").String()
	clientAddr = cli.Flag("endpoint", "Address and port to connect to when in client mode").Default("http://blobstore.kiloe.net").String()

	server             = cli.Command("start", "Start HTTP API service")
	serverAddr         = server.Flag("listen", "Address and port to listen on").Default(":7000").String()
	serverStateDir     = server.Flag("state", "Path to state dir where blobs will be stored").Default("/var/state").ExistingDir()
	serverMaxUploadMem = server.Flag("max-memory", "Megabytes allowed for file uploads before buffering to disk").Default("32").Int64()
	serverMaxBlobSize  = server.Flag("max-size", "Megabyte limit on blob size").Default("128").Int64()

	put      = cli.Command("put", "Store file in the blobstore")
	putFiles = put.Arg("files", "Path to upload to blobstore").Required().ExistingFile()

	get   = cli.Command("get", "Fetch blob data by ID")
	getID = get.Arg("id", "ID of blob to fetch").Required().String()

	info   = cli.Command("info", "Fetch blob info by ID")
	infoID = info.Arg("id", "ID of blob to fetch info for").Required().String()
)

const (
	MB = 1000000
)

func Main() error {
	switch kingpin.MustParse(cli.Parse(os.Args[1:])) {
	case server.FullCommand():
		return ListenAndServe(*serverAddr)
	default:
		return errors.New("not implemented")
	}
}

func main() {
	if err := Main(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
