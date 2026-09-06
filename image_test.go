// This test needs the unexported dockertest resource to see which image a
// backing service actually runs, so it lives in the package under test.
//
//nolint:testpackage
package eltest

import "testing"

// TestContainerImages checks that the backing services run the image tag
// they were created with. A service that forgets to store its tag hands
// dockertest an empty RunOptions.Tag, which resolves to ":latest" instead
// of the pinned release without failing anything.
//
// The tags are the ones the round-trip tests use, so the containers are
// shared with them rather than started a second time.
func TestContainerImages(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		image func(t T) string
	}{
		{
			name: "postgres",
			want: "postgres:" + Postgres17_6,
			image: func(t T) string {
				return NewPostgres(t, Postgres17_6).res.Container.Config.Image
			},
		},
		{
			name: "minio",
			want: "minio/minio:" + Minio202509,
			image: func(t T) string {
				return NewMinio(t, Minio202509).res.Container.Config.Image
			},
		},
		{
			name: "opensearch",
			want: "ghcr.io/ttab/opensearch-icu:" + OpenSearch2_19,
			image: func(t T) string {
				return NewOpenSearch(t, OpenSearch2_19).res.Container.Config.Image
			},
		},
		{
			name: "valkey",
			want: ValkeyRepo + ":" + Valkey8_0,
			image: func(t T) string {
				return NewValkey(t, Valkey8_0).res.Container.Config.Image
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.image(t)
			if got != test.want {
				t.Errorf("running image is %q, expected %q",
					got, test.want)
			}
		})
	}
}
