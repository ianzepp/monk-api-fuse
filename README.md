# monk-api-fuse

A FUSE (Filesystem in Userspace) implementation that mounts the Monk File API as a local filesystem, enabling standard Unix tools (ls, cat, grep, find, etc.) to work with tenant schemas and records.

## Features

- **Direct HTTP/JSON communication** with Monk File API
- **Optimized bandwidth usage** with `?pick=` parameter (30-80% reduction)
- **Metadata caching** for improved performance
- **Read-only support** (Phase 1 POC)
- **Native Go implementation** using go-fuse v2

## Prerequisites

### macOS

1. **Install macFUSE** (required for FUSE support on macOS):
   ```bash
   brew install --cask macfuse
   ```

   After installation, you may need to:
   - Restart your Mac
   - Allow the macFUSE kernel extension in System Settings > Privacy & Security

2. **Install Go** (if not already installed):
   ```bash
   brew install go
   ```

## Installation

```bash
# Clone or navigate to the project
cd ~/Workspaces/monk-api-fuse

# Build the binary
go build -o monk-fuse ./cmd/monk-fuse

# Optional: Install to PATH
sudo cp monk-fuse /usr/local/bin/
```

## Usage

### Quick Start

```bash
# Get your authentication token
export MONK_TOKEN=$(monk auth token)

# Create a mount point
mkdir -p ~/monk-data

# Mount the filesystem
./monk-fuse mount ~/monk-data

# In another terminal, explore the mounted filesystem
ls ~/monk-data
ls ~/monk-data/data
ls ~/monk-data/describe

# Unmount when done
./monk-fuse unmount ~/monk-data
```

### Mount Options

```bash
monk-fuse mount [options] MOUNTPOINT

Options:
  --api-url URL     Monk API base URL (default: http://localhost:8000)
  --token TOKEN     JWT authentication token (or set MONK_TOKEN env var)
  --debug           Enable FUSE debug logging
```

### Examples

```bash
# Mount with explicit token
monk-fuse mount --token eyJhbGc... ~/monk-data

# Mount with custom API URL
monk-fuse mount --api-url https://api.example.com ~/monk-data

# Mount with debug logging
monk-fuse mount --debug ~/monk-data

# Explore mounted data
cd ~/monk-data
ls data/
ls data/issues/
cat data/issues/04b9ce5f-fbc8-4b1a-98c0-79cc99b9c8df/assignee
```

## Architecture

### Performance Optimizations

All FUSE operations leverage the `?pick=` parameter for bandwidth reduction:

| Operation | Pick Parameter | Bandwidth Savings |
|-----------|----------------|-------------------|
| `readdir()` | `?pick=entries` | 60% reduction |
| `getattr()` | `?pick=file_metadata` | 40-50% reduction |
| `read()` | `?pick=content` | 80% reduction |

### Directory Structure

```
monk-api-fuse/
├── cmd/
│   └── monk-fuse/          # CLI entry point
├── pkg/
│   ├── monkapi/            # File API client
│   └── monkfs/             # FUSE filesystem implementation
├── internal/
│   └── cache/              # Metadata cache
└── README.md
```

## Implementation Status

### ✅ Phase 1: Basic Read-Only (Completed)

- [x] Project setup and dependencies
- [x] API client with `?pick=` parameter support
- [x] Basic FUSE operations (Readdir, Getattr, Open, Read)
- [x] HTTP error → errno mapping
- [x] Simple metadata cache
- [x] Mount/unmount CLI commands

### 🚧 Phase 2: Write Support (Not Yet Implemented)

- [ ] Write operations (Write, Create, Truncate)
- [ ] Delete operations (Unlink, Rmdir)
- [ ] Cache invalidation on writes

### 🚧 Phase 3-5: Advanced Features (Future)

- [ ] Advanced caching (LRU, TTL tuning)
- [ ] Prefetching and read-ahead
- [ ] Extended attributes (xattr)
- [ ] Transaction support

## Troubleshooting

### macFUSE not installed

```bash
Error: ... macFUSE kernel extension ...
Solution: Install macFUSE and restart your Mac
brew install --cask macfuse
```

### Permission denied

```bash
Error: permission denied
Solution: Make sure your JWT token is valid
monk auth status
```

### Mount point busy

```bash
Error: mount point busy
Solution: Unmount first
./monk-fuse unmount ~/monk-data
# Or force unmount
umount -f ~/monk-data
```

## Development

```bash
# Run tests (when implemented)
go test ./...

# Build
go build -o monk-fuse ./cmd/monk-fuse

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy
```

## References

- [FUSE.md](../FUSE.md) - Complete specification and design document
- [go-fuse](https://github.com/hanwen/go-fuse) - FUSE library for Go
- [macFUSE](https://osxfuse.github.io/) - FUSE for macOS
- [Monk File API docs](http://localhost:8000/api/docs) - File API documentation

## License

MIT
