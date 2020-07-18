package main

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

func isGSPath(path string) bool {
	return strings.Index(path, "gs://") == 0
}

func parseGSPath(path string) (string, string) {
	path = strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(path, "/", 2)
	return parts[0], parts[1]
}

func doCopy(srcPath, dstPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	var src io.ReadCloser

	if isGSPath(srcPath) {
		bucket, key := parseGSPath(srcPath)
		src, err = client.Bucket(bucket).Object(key).NewReader(ctx)
		if err != nil {
			panic(err)
		}
	} else if srcPath == "-" {
		src = os.Stdin
	} else {
		if src, err = os.Open(srcPath); err != nil {
			panic(err)
		}
	}

	if isGSPath(dstPath) {
		bucket, key := parseGSPath(dstPath)
		writer := client.Bucket(bucket).Object(key).NewWriter(ctx)
		defer writer.Close()

		if _, err := io.Copy(writer, src); err != nil {
			panic(err)
		}
	} else if dstPath == "-" {
		if _, err := io.Copy(os.Stdout, src); err != nil {
			panic(err)
		}
	} else {
		if dstPath == "." {
			parts := strings.Split(srcPath, "/")
			dstPath = parts[len(parts)-1]
		}

		writer, err := os.Create(dstPath)
		if err != nil {
			panic(err)
		}
		defer writer.Close()

		if _, err := io.Copy(writer, src); err != nil {
			panic(err)
		}

	}

}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		panic("missing command")
	}

	switch args[0] {
	case "cp":
		if len(args) < 3 {
			panic("gsutil cp <src> <dst>")
		}
		doCopy(args[1], args[2])

	}
}
