package eltest

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type BackingService interface {
	SetUp(pool *dockertest.Pool, network *dockertest.Network) error
	Purge(pool *dockertest.Pool) error
}

type T interface {
	Name() string
	Helper()
	Fatalf(format string, args ...any)
	Cleanup(fn func())
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

	networks, err := b.pool.NetworksByName("eltest")
	if err != nil {
		return nil, fmt.Errorf("check for eltest network: %w", err)
	}

	if len(networks) > 0 {
		network := networks[0]
		b.network = &network
	} else {
		network, err := b.pool.CreateNetwork("eltest")
		if err != nil {
			return nil, fmt.Errorf("create eltest network: %w", err)
		}

		b.network = network
	}

	for _, c := range b.network.Network.IPAM.Config {
		if c.Gateway == "" {
			continue
		}

		b.gwIP = c.Gateway

		break
	}

	return &b, nil
}

type backingServices struct {
	pool    *dockertest.Pool
	network *dockertest.Network
	gwIP    string

	srvMutex sync.Mutex
	services map[string]BackingService
}

func GetNetwork() (*docker.Network, error) {
	bs, err := getBackingServices()
	if err != nil {
		return nil, err
	}

	return bs.network.Network, nil
}

func GetGatewayIP() (net.IP, error) {
	bs, err := getBackingServices()
	if err != nil {
		return nil, err
	}

	return net.ParseIP(bs.gwIP), nil
}

func Bootstrap[S BackingService](id string, srv S) (S, error) {
	return BootstrapService(id, srv, nil)
}

func BootstrapService[S BackingService](id string, srv S, localTo T) (S, error) {
	var zero S

	bs, err := getBackingServices()
	if err != nil {
		return zero, err
	}

	// Containers that are local to a test are cleaned up with the test
	// instead, don't add to bs.services.
	if localTo != nil {
		err = srv.SetUp(bs.pool, bs.network)
		if err != nil {
			return zero, fmt.Errorf("setup failed: %w", err)
		}

		localTo.Cleanup(func() {
			err := srv.Purge(bs.pool)
			if err != nil {
				localTo.Fatalf("clean up service %s: %w", id, err)
			}
		})

		return srv, nil
	}

	bs.srvMutex.Lock()
	defer bs.srvMutex.Unlock()

	existing, ok := bs.services[id]
	if ok {
		s, ok := existing.(S)
		if !ok {
			return zero, fmt.Errorf("type mismatch, expected %T got %T",
				srv, existing)
		}

		return s, nil
	}

	err = srv.SetUp(bs.pool, bs.network)
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

	bs.services = make(map[string]BackingService)

	return errors.Join(errs...)
}
