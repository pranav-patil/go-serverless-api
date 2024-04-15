package util

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/md5"  //nolint:gosec // even though md5 is insecure we still want to use it to obfuscate userId in the path.
	"crypto/sha1" //nolint:gosec // even though sha1 is insecure we still want to use it for data integrity of bookmark pkge
	"encoding/hex"
	"io"
	"time"

	"github.com/rs/zerolog/log"
)

func Compress(payload string) ([]byte, error) {
	var err error
	var b bytes.Buffer
	gzWriter := gzip.NewWriter(&b)
	gzWriter.Comment = "Emprovise bookmarks data. Private and confidential."

	if _, err = gzWriter.Write([]byte(payload)); err != nil {
		log.Error().Msgf("Gzip Compress Write Error: %v", err.Error())
	}
	if err = gzWriter.Close(); err != nil {
		log.Error().Msgf("Gzip Writer Close Error: %v", err.Error())
	}

	return b.Bytes(), err
}

func Decompress(payload []byte) (string, error) {
	reader := bytes.NewReader(payload)
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		log.Error().Msgf("Gzip Decompress NewReader Error: %v", err.Error())
		return "", err
	}

	output, err := io.ReadAll(gzReader)
	if err != nil {
		log.Error().Msgf("Gzip Decompress ReadAll Error: %v", err.Error())
		return "", err
	}

	return string(output), nil
}

func ByteCompress(payload []byte) ([]byte, error) {
	var err error
	var b bytes.Buffer
	gzWriter := gzip.NewWriter(&b)
	gzWriter.Comment = "Emprovise bookmarks data. Private and confidential."

	if _, err = gzWriter.Write(payload); err != nil {
		log.Error().Msgf("ByteCompress Gzip Write Error: %v", err.Error())
	}
	if err = gzWriter.Close(); err != nil {
		log.Error().Msgf("ByteCompress Gzip Writer Close Error: %v", err.Error())
	}

	return b.Bytes(), err
}

func CreateZipFile(files map[string]string) (res []byte, err error) {
	buffer := new(bytes.Buffer)
	writer := zip.NewWriter(buffer)

	for filename, content := range files {
		var f io.Writer
		f, err = writer.Create(filename)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func CreateTarFile(files map[string]string) (res []byte, err error) {
	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)

	for filename, content := range files {
		data := []byte(content)

		header := &tar.Header{
			Name:     filename,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
			Mode:     0o755,
			ModTime:  time.Now(),
		}

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return nil, err
		}

		_, err = tarWriter.Write(data)
		if err != nil {
			return nil, err
		}
	}

	err = tarWriter.Close()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func SHA1Checksum(data []byte) string {
	hasher := sha1.New() //nolint:gosec // even though sha1 is insecure we still want to use it for data integrity of bookmark pkge
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func MD5Hash(data string) string {
	hasher := md5.New() //nolint:gosec // even though md5 is insecure we still want to use it to obfuscate userId in the path.
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
