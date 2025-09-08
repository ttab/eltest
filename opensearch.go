package eltest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

func NewOpenSearch(t T, tag string) *OpenSearch {
	os, err := Bootstrap("opensearch-"+tag, &OpenSearch{
		tag: tag,
	})
	Must(t, err, "bootstrap opensearch "+tag)

	return os
}

type OpenSearch struct {
	res *dockertest.Resource
	tag string
}

func (m *OpenSearch) GetEndpoint() string {
	return fmt.Sprintf("http://%s:9200",
		m.res.Container.NetworkSettings.IPAddress)
}

func (m *OpenSearch) SetUp(pool *dockertest.Pool) error {
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "opensearchproject/opensearch",
		Tag:        m.tag,
		Env: []string{
			"discovery.type=single-node",
			"OPENSEARCH_INITIAL_ADMIN_PASSWORD=admin",
			"DISABLE_INSTALL_DEMO_CONFIG=true",
			"DISABLE_SECURITY_PLUGIN=true",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return fmt.Errorf("failed to run minio container: %w", err)
	}

	m.res = res

	// Make sure that containers don't stick around for more than an hour,
	// even if in-process cleanup fails.
	_ = res.Expire(3600)

	endpoint := m.GetEndpoint()

	err = pool.Retry(func() error {
		res, err := http.Get(endpoint)
		if err != nil {
			log.Println(err.Error())

			return fmt.Errorf("request instance details: %w", err)
		}

		err = res.Body.Close()
		if err != nil {
			return fmt.Errorf("close response body: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("error response: %s", res.Status)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to connect to minio: %w", err)
	}

	return nil
}

func (m *OpenSearch) Purge(pool *dockertest.Pool) error {
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
