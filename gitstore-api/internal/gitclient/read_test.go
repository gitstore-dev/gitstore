// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2026 GitStore contributors

// Unit tests for gRPC read client methods.
// Uses bufconn to run a real gRPC server in-process — no Docker required.

package gitclient_test

import (
	"context"
	"net"
	"testing"

	gitv1 "github.com/gitstore-dev/gitstore/api/gen/gitstore/git/v1"
	"github.com/gitstore-dev/gitstore/api/internal/gitclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// stubGitServer is a minimal gRPC server for testing.
type stubGitServer struct {
	gitv1.UnimplementedGitServiceServer

	getFileFunc      func(*gitv1.GetFileRequest) (*gitv1.GetFileResponse, error)
	listFilesFunc    func(*gitv1.ListFilesRequest) (*gitv1.ListFilesResponse, error)
	getLatestTagFunc func(*gitv1.GetLatestTagRequest) (*gitv1.GetLatestTagResponse, error)
	listTagsFunc     func(*gitv1.ListTagsRequest) (*gitv1.ListTagsResponse, error)
}

func (s *stubGitServer) GetFile(_ context.Context, req *gitv1.GetFileRequest) (*gitv1.GetFileResponse, error) {
	if s.getFileFunc != nil {
		return s.getFileFunc(req)
	}
	return nil, status.Error(codes.Unimplemented, "not set up")
}

func (s *stubGitServer) ListFiles(_ context.Context, req *gitv1.ListFilesRequest) (*gitv1.ListFilesResponse, error) {
	if s.listFilesFunc != nil {
		return s.listFilesFunc(req)
	}
	return nil, status.Error(codes.Unimplemented, "not set up")
}

func (s *stubGitServer) GetLatestTag(_ context.Context, req *gitv1.GetLatestTagRequest) (*gitv1.GetLatestTagResponse, error) {
	if s.getLatestTagFunc != nil {
		return s.getLatestTagFunc(req)
	}
	return nil, status.Error(codes.Unimplemented, "not set up")
}

func (s *stubGitServer) ListTags(_ context.Context, req *gitv1.ListTagsRequest) (*gitv1.ListTagsResponse, error) {
	if s.listTagsFunc != nil {
		return s.listTagsFunc(req)
	}
	return nil, status.Error(codes.Unimplemented, "not set up")
}

// newBufconnClient starts a bufconn gRPC server with the given stub and returns a Client.
func newBufconnClient(t *testing.T, stub *stubGitServer) *gitclient.Client {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	gitv1.RegisterGitServiceServer(srv, stub)

	go func() {
		if err := srv.Serve(lis); err != nil && err.Error() != "closed" {
			t.Logf("bufconn server error: %v", err)
		}
	}()
	t.Cleanup(func() {
		srv.Stop()
		lis.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return gitclient.NewClientFromConn(conn)
}

func TestReadFile_OK(t *testing.T) {
	want := []byte("---\nid: p1\n---\nhello")
	stub := &stubGitServer{
		getFileFunc: func(req *gitv1.GetFileRequest) (*gitv1.GetFileResponse, error) {
			assert.Equal(t, "products/p1.md", req.Path)
			assert.Equal(t, "v1.0.0", req.Ref)
			return &gitv1.GetFileResponse{Content: want}, nil
		},
	}
	c := newBufconnClient(t, stub)

	got, err := c.ReadFile(context.Background(), "products/p1.md", "v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestReadFile_NotFound(t *testing.T) {
	stub := &stubGitServer{
		getFileFunc: func(_ *gitv1.GetFileRequest) (*gitv1.GetFileResponse, error) {
			return nil, status.Error(codes.NotFound, "file not found")
		},
	}
	c := newBufconnClient(t, stub)

	_, err := c.ReadFile(context.Background(), "missing.md", "HEAD")
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestListFiles_OK(t *testing.T) {
	stub := &stubGitServer{
		listFilesFunc: func(req *gitv1.ListFilesRequest) (*gitv1.ListFilesResponse, error) {
			assert.Equal(t, "products/", req.PathPrefix)
			assert.Equal(t, "v1.0.0", req.Ref)
			return &gitv1.ListFilesResponse{
				Files: []*gitv1.FileEntry{
					{Path: "products/a.md", SizeBytes: 100},
					{Path: "products/b.md", SizeBytes: 200},
				},
			}, nil
		},
	}
	c := newBufconnClient(t, stub)

	entries, err := c.ListFiles(context.Background(), "products/", "v1.0.0")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "products/a.md", entries[0].Path)
}

func TestGetLatestTag_OK(t *testing.T) {
	stub := &stubGitServer{
		getLatestTagFunc: func(_ *gitv1.GetLatestTagRequest) (*gitv1.GetLatestTagResponse, error) {
			return &gitv1.GetLatestTagResponse{
				Tag:   &gitv1.TagEntry{Name: "v2.3.1", CommitSha: "abc123"},
				Found: true,
			}, nil
		},
	}
	c := newBufconnClient(t, stub)

	tag, err := c.GetLatestTag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v2.3.1", tag.Name)
	assert.Equal(t, "abc123", tag.CommitSha)
}

func TestGetLatestTag_NotFound(t *testing.T) {
	stub := &stubGitServer{
		getLatestTagFunc: func(_ *gitv1.GetLatestTagRequest) (*gitv1.GetLatestTagResponse, error) {
			return &gitv1.GetLatestTagResponse{Found: false}, nil
		},
	}
	c := newBufconnClient(t, stub)

	_, err := c.GetLatestTag(context.Background())
	require.Error(t, err)
}

func TestListTags_OK(t *testing.T) {
	stub := &stubGitServer{
		listTagsFunc: func(req *gitv1.ListTagsRequest) (*gitv1.ListTagsResponse, error) {
			assert.Equal(t, "v", req.Prefix)
			return &gitv1.ListTagsResponse{
				Tags: []*gitv1.TagEntry{
					{Name: "v1.0.0", CommitSha: "sha1"},
					{Name: "v1.1.0", CommitSha: "sha2"},
				},
			}, nil
		},
	}
	c := newBufconnClient(t, stub)

	tags, err := c.ListTags(context.Background(), "v")
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, "v1.0.0", tags[0].Name)
}
