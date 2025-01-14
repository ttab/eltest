package eltest

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ory/dockertest/v3"
)

type BackingService interface {
	SetUp(pool *dockertest.Pool) error
	Purge(pool *dockertest.Pool) error
}

type T interface {
	Name() string
	Helper()
	Fatalf(format string, args ...any)
}

func Must(t T, err error, actionFormat string, a ...any) {
	t.Helper()

	action := fmt.Sprintf(actionFormat, a...)

	if err != nil {
		t.Fatalf("failed: %s: %v", action, err)
	}
}

func PurgeBackingServices() error {
	backingM.Lock()
	defer backingM.Unlock()

	if back == nil {
		return nil
	}

	err := back.Purge()

	back = nil

	return err
}

var (
	backingM sync.Mutex
	back     *backingServices
	bErr     error
)

func getBackingServices() (*backingServices, error) {
	backingM.Lock()
	defer backingM.Unlock()

	if back != nil || bErr != nil {
		return back, bErr
	}

	back, bErr = createBackingServices()

	return back, bErr
}

func createBackingServices() (*backingServices, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to create docker pool: %w", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	b := backingServices{
		pool:     pool,
		services: make(map[string]BackingService),
	}

	return &b, nil
}

type backingServices struct {
	pool *dockertest.Pool

	srvMutex sync.Mutex
	services map[string]BackingService
}

func Bootstrap[T BackingService](id string, srv T) (T, error) {
	var zero T

	bs, err := getBackingServices()
	if err != nil {
		return zero, err
	}

	bs.srvMutex.Lock()
	defer bs.srvMutex.Unlock()

	existing, ok := bs.services[id]
	if ok {
		s, ok := existing.(T)
		if !ok {
			return zero, fmt.Errorf("type mismatch, expected %T got %T",
				srv, existing)
		}

		return s, nil
	}

	err = srv.SetUp(bs.pool)
	if err != nil {
		return zero, fmt.Errorf("setup failed: %w", err)
	}

	bs.services[id] = srv

	return srv, nil
}

func (bs *backingServices) Purge() error {
	bs.srvMutex.Lock()
	defer bs.srvMutex.Unlock()

	var errs []error

	for id, srv := range bs.services {
		err := srv.Purge(bs.pool)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
	}

	return errors.Join(errs...)
}
