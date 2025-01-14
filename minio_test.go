package eltest_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/ttab/eltest"
)

func TestMinio(t *testing.T) {
	minioSvc := eltest.NewMinio(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	client, err := minioSvc.Client()
	eltest.Must(t, err, "get minio client")

	bucket := minioSvc.CreateBucket(t, ctx, "some-bucket")

	data := "world"

	_, err = client.PutObject(ctx, bucket, "hello.txt",
		strings.NewReader(data),
		int64(len(data)), minio.PutObjectOptions{})
	eltest.Must(t, err, "create object")

	obj, err := client.GetObject(ctx, bucket, "hello.txt", minio.GetObjectOptions{})
	eltest.Must(t, err, "get object")

	defer func() {
		err := obj.Close()
		eltest.Must(t, err, "close test object response")
	}()

	gotData, err := io.ReadAll(obj)
	eltest.Must(t, err, "read object response")

	got := string(gotData)
	if got != data {
		t.Fatalf("got %q back, expected %q", got, data)
	}
}
