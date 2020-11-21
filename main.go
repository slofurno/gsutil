package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

func init() {
	certs := x509.NewCertPool()
	if ok := certs.AppendCertsFromPEM(getAlpineCerts()); !ok {
		panic("failed to read certs")
	}

	tlsConfig := &tls.Config{
		RootCAs: certs,
	}

	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}
}

func getAlpineCerts() []byte {
	decoded, err := base64.StdEncoding.DecodeString(AlpineCerts)
	if err != nil {
		panic(err)
	}

	br := bytes.NewReader(decoded)

	gz, err := gzip.NewReader(br)
	if err != nil {
		panic(err)
	}
	defer gz.Close()

	b, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(err)
	}

	return b
}

func isGSPath(path string) bool {
	return strings.Index(path, "gs://") == 0
}

func parseGSPath(path string) (string, string) {
	path = strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(path, "/", 2)
	return parts[0], parts[1]
}

var units = []string{"B", "KiB", "MiB", "GiB", "TiB"}

func humanSize(size int64) string {

	for i := 0; i < len(units); i++ {
		n := size >> 10
		if n == 0 {
			return fmt.Sprintf("%d %s", size, units[i])
		}
		size = n
	}
	return ""
}

func list(path string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	bucket, key := parseGSPath(path)
	h := client.Bucket(bucket)

	it := h.Objects(ctx, &storage.Query{
		Prefix: key,
	})

	for {
		oa, err := it.Next()
		if err != nil {
			return
		}

		fmt.Printf("%s %s %s\n", humanSize(oa.Size), oa.Updated.Format(time.RFC3339), oa.Name)
	}
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

		if _, err := io.Copy(writer, src); err != nil {
			panic(err)
		}

		if err := writer.Close(); err != nil {
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

		if _, err := io.Copy(writer, src); err != nil {
			panic(err)
		}

		if err := writer.Close(); err != nil {
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
	case "ls":
		list(args[1])
	}
}
