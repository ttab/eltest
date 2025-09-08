package eltest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ttab/eltest"
)

func TestOpenSearch(t *testing.T) {
	svc := eltest.NewOpenSearch(t, "2.19.0")

	endpoint := svc.GetEndpoint()

	res, err := http.Get(endpoint)
	eltest.Must(t, err, "request instance details")

	defer func() {
		err := res.Body.Close()
		eltest.Must(t, err, "close response body")
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("error response: %s", res.Status)
	}

	var response struct {
		Version struct {
			Distribution string `json:"distribution"`
			Number       string `json:"number"`
		} `json:"version"`
	}

	dec := json.NewDecoder(res.Body)

	err = dec.Decode(&response)
	eltest.Must(t, err, "decode response body")

	if response.Version.Distribution != "opensearch" {
		t.Errorf(`expected distribution to be "opensearch", got %q`,
			response.Version.Distribution)
	}

	if response.Version.Number != "2.19.0" {
		t.Errorf(`expected version number to be "2.19.0", got %q`,
			response.Version.Number)
	}
}
