# eltest - Elephant test

Support library for writing elephant tests.

Takes care of setting up Postgres and Minio containers for intergration tests. See tests for usage, and don't miss setting up `TestMain()` in your own service:

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
