package monkfs

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/ianzepp/monk-api-fuse/internal/cache"
	"github.com/ianzepp/monk-api-fuse/pkg/monkapi"
)

// parseMonkTimestamp converts ISO 8601 (RFC3339) format to Unix timestamp
func parseMonkTimestamp(ts string) uint64 {
	if ts == "" {
		return 0
	}
	// Parse ISO 8601: 2025-11-17T19:26:40Z
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	return uint64(t.Unix())
}

// MonkFS implements the FUSE filesystem interface
type MonkFS struct {
	fs.Inode
	apiClient *monkapi.Client
	cache     *cache.MetadataCache
}

// NewMonkFS creates a new Monk FUSE filesystem
func NewMonkFS(apiClient *monkapi.Client) *MonkFS {
	return &MonkFS{
		apiClient: apiClient,
		cache:     cache.NewMetadataCache(30 * time.Second),
	}
}

var _ = (fs.NodeReaddirer)((*MonkFS)(nil))
var _ = (fs.NodeGetattrer)((*MonkFS)(nil))
var _ = (fs.NodeOpener)((*MonkFS)(nil))
var _ = (fs.NodeLookuper)((*MonkFS)(nil))

// Readdir implements directory listing
func (n *MonkFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	path := n.getPath()

	// Use pick=entries to get just the array (60% bandwidth reduction)
	resp, err := n.apiClient.List(ctx, path, monkapi.ListOptions{
		LongFormat: true,
	}, "entries")
	if err != nil {
		return nil, HTTPErrorToErrno(err)
	}

	entries := []fuse.DirEntry{}
	for _, entry := range resp.Entries {
		mode := parseFileMode(entry.FilePermissions, entry.FileType)
		entries = append(entries, fuse.DirEntry{
			Name: entry.Name,
			Mode: mode,
			Ino:  hashPath(entry.Path),
		})
	}

	return fs.NewListDirStream(entries), 0
}

// Getattr implements stat() functionality
func (n *MonkFS) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	path := n.getPath()

	// Check cache first
	if cached := n.cache.Get(path); cached != nil {
		fillAttr(&out.Attr, cached)
		return 0
	}

	// Use pick=file_metadata to get only metadata (40-50% bandwidth reduction)
	resp, err := n.apiClient.Stat(ctx, path, "file_metadata")
	if err != nil {
		if monkapi.IsNotFound(err) {
			return syscall.ENOENT
		}
		return HTTPErrorToErrno(err)
	}

	// Cache the result
	n.cache.Set(path, resp)

	fillAttr(&out.Attr, resp)
	return 0
}

// Lookup looks up a child node by name
func (n *MonkFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	path := n.getPath() + "/" + name

	resp, err := n.apiClient.Stat(ctx, path, "file_metadata")
	if err != nil {
		if monkapi.IsNotFound(err) {
			return nil, syscall.ENOENT
		}
		return nil, HTTPErrorToErrno(err)
	}

	// Cache the result
	n.cache.Set(path, resp)

	// Create child inode
	child := n.NewInode(ctx, &MonkFS{
		apiClient: n.apiClient,
		cache:     n.cache,
	}, fs.StableAttr{
		Mode: parseStatMode(resp),
		Ino:  hashPath(path),
	})

	fillAttr(&out.Attr, resp)
	return child, 0
}

// Open implements file open
func (n *MonkFS) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	path := n.getPath()

	// Validate file exists (pick="" for minimal validation)
	_, err := n.apiClient.Stat(ctx, path, "")
	if err != nil {
		if monkapi.IsNotFound(err) {
			return nil, 0, syscall.ENOENT
		}
		return nil, 0, HTTPErrorToErrno(err)
	}

	return &MonkFileHandle{
		node: n,
		path: path,
	}, fuse.FOPEN_KEEP_CACHE, 0
}

// MonkFileHandle represents an open file handle
type MonkFileHandle struct {
	node *MonkFS
	path string
}

var _ = (fs.FileReader)((*MonkFileHandle)(nil))

// Read implements file reading
func (fh *MonkFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// Use pick=content to get just the file content (80% reduction for single fields!)
	resp, err := fh.node.apiClient.Retrieve(ctx, fh.path, monkapi.RetrieveOptions{
		StartOffset: int(off),
		MaxBytes:    len(dest),
	}, "content")
	if err != nil {
		return nil, HTTPErrorToErrno(err)
	}

	// Convert content to bytes
	data := contentToBytes(resp.Content)

	// Handle offset
	if off >= int64(len(data)) {
		return fuse.ReadResultData([]byte{}), 0
	}

	return fuse.ReadResultData(data[off:]), 0
}

// Helper functions

func (n *MonkFS) getPath() string {
	path := n.Path(nil)
	if path == "" {
		return "/"
	}
	return "/" + path
}

func parseFileMode(permissions string, fileType string) uint32 {
	mode := uint32(0)

	if fileType == "d" {
		mode |= syscall.S_IFDIR | 0755
	} else {
		mode |= syscall.S_IFREG | 0644
	}

	return mode
}

func parseStatMode(stat *monkapi.StatResponse) uint32 {
	if stat.Type == "directory" || stat.FileMetadata.Type == "directory" {
		return syscall.S_IFDIR | 0755
	}
	return syscall.S_IFREG | 0644
}

func fillAttr(attr *fuse.Attr, stat *monkapi.StatResponse) {
	attr.Size = uint64(stat.FileMetadata.Size)
	attr.Mtime = parseMonkTimestamp(stat.FileMetadata.ModifiedTime)
	attr.Ctime = parseMonkTimestamp(stat.FileMetadata.CreatedTime)
	attr.Atime = parseMonkTimestamp(stat.FileMetadata.AccessTime)

	if stat.Type == "directory" || stat.FileMetadata.Type == "directory" {
		attr.Mode = syscall.S_IFDIR | 0755
	} else {
		attr.Mode = syscall.S_IFREG | 0644
	}
}

func hashPath(path string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(path))
	return h.Sum64()
}

func contentToBytes(content interface{}) []byte {
	if content == nil {
		return []byte{}
	}

	switch v := content.(type) {
	case string:
		// Remove JSON quotes if present (pick returns valid JSON)
		if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
			v = v[1 : len(v)-1]
		}
		return []byte(v)
	case []byte:
		return v
	default:
		// Convert to JSON
		data, _ := json.Marshal(v)
		return data
	}
}
