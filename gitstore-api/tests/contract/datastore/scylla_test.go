// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2026 GitStore contributors

//go:build scylla

// Wires the contract suite against the ScyllaDB backend using testcontainers.
// A single container is started in TestMain and shared across all tests.

package datastore_contract_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gitstore-dev/gitstore/api/internal/config"
	"github.com/gitstore-dev/gitstore/api/internal/datastore"
	"github.com/gitstore-dev/gitstore/api/internal/datastore/scylla"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// scyllaContainerAddr is set by TestMain before any test runs.
var scyllaContainerAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "scylladb/scylla:5.4",
		ExposedPorts: []string{"9042/tcp"},
		Cmd:          []string{"--developer-mode=1", "--overprovisioned=1", "--smp=1"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("9042/tcp"),
			wait.ForLog("Starting listening for CQL clients").
				WithStartupTimeout(120*time.Second),
		),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic("failed to start ScyllaDB container: " + err.Error())
	}

	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "9042")
	scyllaContainerAddr = host + ":" + port.Port()

	code := m.Run()
	_ = c.Terminate(ctx)
	os.Exit(code)
}

func newScyllaDatastore(t *testing.T) datastore.Datastore {
	t.Helper()
	cfg := config.ScyllaConfig{
		Hosts:    []string{scyllaContainerAddr},
		Keyspace: "gitstore",
	}
	store, err := scylla.New(cfg, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestContractScylla(t *testing.T) {
	RunContractSuite(t, newScyllaDatastore(t))
}
