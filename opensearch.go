package eltest

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const OpenSearch2_19 = "v2.19.3-2"

func NewOpenSearch(t T, tag string) *OpenSearch {
	os, err := Bootstrap("opensearch-"+tag, &OpenSearch{
		tag: tag,
	})
	Must(t, err, "bootstrap opensearch %s", tag)

	return os
}

type OpenSearch struct {
	res *dockertest.Resource
	tag string
}

func (m *OpenSearch) GetEndpoint() string {
	return fmt.Sprintf("http://localhost:%s", m.res.GetPort("9200/tcp"))
}

func (m *OpenSearch) GetContainerEndpoint() string {
	hostname := strings.TrimPrefix(m.res.Container.Name, "/")

	return fmt.Sprintf("http://%s:9200", hostname)
}

func (m *OpenSearch) SetUp(pool *dockertest.Pool, network *dockertest.Network) error {
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "ghcr.io/ttab/opensearch-icu",
		Tag:        m.tag,
		Env: []string{
			"discovery.type=single-node",
			"OPENSEARCH_INITIAL_ADMIN_PASSWORD=admin",
			"DISABLE_INSTALL_DEMO_CONFIG=true",
			"DISABLE_SECURITY_PLUGIN=true",
		},
		NetworkID: network.Network.ID,
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return fmt.Errorf("failed to run opensearch container: %w", err)
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
		return fmt.Errorf("failed to connect to opensearch: %w", err)
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
			"failed to purge opensearch container: %w", err)
	}

	return nil
}
