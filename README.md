# eltest - Elephant test

Support library for writing elephant tests.

Takes care of setting up Postgres, Minio, OpenSearch and Valkey containers for integration tests. See tests for usage, and don't miss setting up `TestMain()` in your own service:

``` go
func TestMain(m *testing.M) {
	code := m.Run()

	err := eltest.PurgeBackingServices()
	if err != nil {
		log.Printf("failed to purge backing services: %v", err)
	}

	os.Exit(code)
}
```

The Postgres version is chosen with one of the exported tag constants, so a
service moves between major versions by changing the constant it passes:

``` go
pg := eltest.NewPostgres(t, eltest.Postgres18_6)
```
