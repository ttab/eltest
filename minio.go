package eltest

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	Minio202302 = "RELEASE.2023-02-22T18-23-45Z"
	Minio202509 = "RELEASE.2025-09-07T16-13-09Z"
)

func NewMinio(t T, tag string) *Minio {
	m, err := Bootstrap("minio-"+tag, &Minio{})
	Must(t, err, "bootstrap minio")

	return m
}

type Minio struct {
	tag string
	res *dockertest.Resource
	ip  string
}

const (
	minioID     = "minioadmin"
	minioSecret = "minioadmin"
)

func (m *Minio) getS3Endpoint() string {
	return fmt.Sprintf("%s:9000", m.ip)
}

type MinioEnvironment struct {
	Endpoint string
	ID       string
	Secret   string
}

func (m *Minio) Client() (*minio.Client, error) {
	svc, err := minio.New(
		m.getS3Endpoint(),
		&minio.Options{
			Creds: credentials.NewStaticV4(
				minioID, minioSecret, "",
			),
			Secure: false,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	return svc, nil
}

func (m *Minio) Environment() MinioEnvironment {
	return MinioEnvironment{
		Endpoint: m.getS3Endpoint(),
		ID:       minioID,
		Secret:   minioSecret,
	}
}

// CreateBucket with the given prefix and a suffix based on the test name.
func (m *Minio) CreateBucket(t T, ctx context.Context, prefix string) string {
	client, err := m.Client()
	Must(t, err, "create client")

	sane := strings.ToLower(sanitizeExp.ReplaceAllString(t.Name(), "-"))

	bucketName := prefix + "-" + sane

	err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	Must(t, err, "create bucket %q", bucketName)

	return bucketName
}

func (m *Minio) SetUp(pool *dockertest.Pool, network *dockertest.Network) error {
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "minio/minio",
		Tag:        m.tag,
		Cmd:        []string{"server", "/data"},
		NetworkID:  network.Network.ID,
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return fmt.Errorf("failed to run minio container: %w", err)
	}

	m.res = res
	m.ip = res.GetIPInNetwork(network)

	// Make sure that containers don't stick around for more than an hour,
	// even if in-process cleanup fails.
	_ = res.Expire(3600)

	client, err := m.Client()
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	err = pool.Retry(func() error {
		_, err = client.ListBuckets(context.Background())
		if err != nil {
			log.Println(err.Error())

			return fmt.Errorf("failed to list buckets: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to connect to minio: %w", err)
	}

	return nil
}

func (m *Minio) Purge(pool *dockertest.Pool) error {
	if m.res == nil {
		return nil
	}

	err := pool.Purge(m.res)
	if err != nil {
		return fmt.Errorf(
			"failed to purge minio container: %w", err)
	}

	return nil
}
