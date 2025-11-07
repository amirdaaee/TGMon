# Stream Package

The `stream` package provides a robust, production-ready solution for streaming Telegram media files. It implements a worker pool pattern with automatic failover, caching, and flood wait handling to efficiently download files from Telegram channels.

## Overview

This package abstracts the complexity of Telegram's file download API, providing:

- **Worker Pool**: Multiple bot accounts working together to distribute load
- **Automatic Failover**: Seamless switching between workers on rate limits
- **Caching**: Disk-backed cache for document metadata and access hashes
- **Range Requests**: Support for partial file downloads (HTTP Range headers)
- **Buffered Streaming**: Efficient `io.Reader` interface for streaming large files
- **Flood Wait Handling**: Automatic retry and worker rotation on Telegram rate limits

## Key Concepts

### Worker (`IWorker`)
A single bot account that can:
- Fetch document metadata from Telegram channels
- Download document thumbnails
- Stream file chunks

### Worker Pool (`IWorkerPool`)
A collection of workers that:
- Distributes requests across workers in round-robin fashion
- Automatically switches to another worker on flood waits
- Creates streamers for downloading files

### Streamer (`IStreamer`)
An `io.Reader` implementation that:
- Streams file content from Telegram
- Handles worker rotation transparently
- Manages buffering for efficient reads

## Setup

### Prerequisites

1. **Telegram Bot Tokens**: One or more bot tokens (more tokens = better resilience)
2. **Telegram App Credentials**: App ID and App Hash from [my.telegram.org](https://my.telegram.org)
3. **Channel ID**: The numeric ID of the Telegram channel containing the media
4. **Session Directory**: Directory for storing Telegram session files
5. **Cache Directory**: Directory for caching document metadata and access hashes

### Configuration

Create a `tlg.SessionConfig` with your Telegram credentials:

```go
import (
    "github.com/amirdaaee/TGMon/internal/tlg"
)

sessCfg := &tlg.SessionConfig{
    AppID:      12345678,                    // Your Telegram App ID
    AppHash:    "your-app-hash-here",        // Your Telegram App Hash
    SessionDir: "./sessions",                // Directory for session files
    SocksProxy: "socks5://user:pass@host:port", // Optional: SOCKS5 proxy
}
```

## Basic Usage

### 1. Initialize Worker Pool

Create a pool with multiple bot tokens for redundancy:

```go
import (
    "github.com/amirdaaee/TGMon/internal/stream"
)

tokens := []string{
    "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz",
    "0987654321:XYZabcDEFghiJKLmnoPQRstu",
    // Add more tokens for better resilience
}

channelID := int64(-1001234567890)  // Your channel ID
cacheRoot := "./storage/cache"      // Cache directory

pool, err := stream.NewWorkerPool(tokens, sessCfg, channelID, cacheRoot)
if err != nil {
    log.Fatalf("Failed to create worker pool: %v", err)
}
```

**Note**: The pool initializes workers concurrently. If all workers fail to initialize, `NewWorkerPool` returns an error.

### 2. Stream a File

Stream a complete file from a message:

```go
ctx := context.Background()
msgID := 12345  // Message ID containing the document

// Stream entire file (offset=0, end=fileSize-1)
streamer, err := pool.Stream(ctx, msgID, 0, fileSize-1)
if err != nil {
    log.Fatalf("Failed to create streamer: %v", err)
}

// Use the buffered reader for efficient reading
reader := streamer.GetBuffer()

// Read file content
data, err := io.ReadAll(reader)
if err != nil {
    log.Fatalf("Failed to read stream: %v", err)
}
```

### 3. Stream a File Range (Partial Download)

Stream a specific byte range (useful for HTTP Range requests):

```go
ctx := context.Background()
msgID := 12345
offset := int64(1024 * 1024)  // Start at 1MB
end := int64(5 * 1024 * 1024) // End at 5MB

streamer, err := pool.Stream(ctx, msgID, offset, end)
if err != nil {
    log.Fatalf("Failed to create streamer: %v", err)
}

reader := streamer.GetBuffer()
// Read only the specified range
data := make([]byte, end-offset+1)
_, err = io.ReadFull(reader, data)
```

### 4. Use as HTTP Handler

The streamer implements `io.Reader`, making it perfect for HTTP responses:

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
    msgID := 12345
    fileSize := int64(10 * 1024 * 1024) // 10MB

    // Parse Range header if present
    offset := int64(0)
    end := fileSize - 1

    if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
        // Parse range header and set offset/end
        // ... (use a range parser library)
    }

    streamer, err := pool.Stream(r.Context(), msgID, offset, end)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "video/mp4")
    w.Header().Set("Content-Length", strconv.FormatInt(end-offset+1, 10))

    // Stream directly to response
    io.Copy(w, streamer.GetBuffer())
}
```

## Advanced Usage

### Get Document Metadata

Retrieve document information without downloading:

```go
worker := pool.GetNextWorker()
doc, err := worker.GetDoc(ctx, msgID)
if err != nil {
    log.Fatalf("Failed to get document: %v", err)
}

fileSize := doc.GetSize()
mimeType := doc.GetMimeType()
fileName := doc.GetFileName()
```

### Get Thumbnail

Download a document's thumbnail:

```go
worker := pool.GetNextWorker()
thumbnail, err := worker.GetThumbnail(ctx, msgID)
if err != nil {
    if errors.Is(err, stream.ErrNoThumbnail) {
        log.Println("Document has no thumbnail")
    } else {
        log.Fatalf("Failed to get thumbnail: %v", err)
    }
}

// Use thumbnail bytes (e.g., save to file, serve via HTTP)
```

### Direct Worker Access

Use a specific worker directly (bypassing pool rotation):

```go
worker := pool.GetNextWorker()

// Use worker methods directly
doc, err := worker.GetDoc(ctx, msgID)
thumbnail, err := worker.GetThumbnail(ctx, msgID)
```

## Error Handling

### Common Errors

- **`stream.ErrNoThumbnail`**: Document has no thumbnail available
- **`downloader.ErrFloodWaitTooLong`**: Flood wait exceeds threshold (handled automatically by pool)
- **`io.EOF`**: End of file reached (normal when streaming completes)

### Error Handling Example

```go
streamer, err := pool.Stream(ctx, msgID, offset, end)
if err != nil {
    // Handle initialization errors
    return fmt.Errorf("failed to create streamer: %w", err)
}

reader := streamer.GetBuffer()
buffer := make([]byte, 8192)

for {
    n, err := reader.Read(buffer)
    if err == io.EOF {
        break // Normal end of stream
    }
    if err != nil {
        // Handle read errors
        return fmt.Errorf("read error: %w", err)
    }

    // Process buffer[:n]
    processChunk(buffer[:n])
}
```

## How It Works

### Worker Pool Architecture

1. **Initialization**: Multiple workers connect to Telegram concurrently
2. **Round-Robin Selection**: Requests are distributed evenly across workers
3. **Automatic Failover**: On flood wait errors, the pool automatically tries the next worker
4. **Caching**: Document metadata and access hashes are cached on disk to reduce API calls

### Streaming Flow

1. **Stream Creation**: `pool.Stream()` creates a `Streamer` with a `downloader.Reader`
2. **Chunk Download**: The reader downloads chunks aligned to 4KB boundaries (Telegram requirement)
3. **Worker Rotation**: If a worker hits a flood wait, the streamer automatically switches workers
4. **Buffering**: Data is buffered for efficient reading
5. **Range Trimming**: Partial ranges are trimmed to exact byte boundaries

### Caching Strategy

- **Document Cache**: Encoded document metadata cached to avoid repeated API calls
- **Access Hash Cache**: Document access hashes cached for faster thumbnail access
- **Cache Location**: Files stored in `{cacheRoot}/{workerID}-{messageID}-{type}`

## Best Practices

1. **Multiple Workers**: Use at least 2-3 bot tokens for redundancy
2. **Context Management**: Always pass a context with timeout for long-running operations
3. **Error Handling**: Check for `io.EOF` separately from other errors
4. **Resource Cleanup**: Let the streamer be garbage collected after use
5. **Cache Directory**: Use a dedicated directory for cache files
6. **Session Directory**: Each worker creates its own session file

## Performance Considerations

- **Chunk Sizes**: The downloader automatically selects optimal chunk sizes (4KB to 512KB)
- **Buffering**: Default buffer size is 8MB (configurable via `RuntimeConfig.StreamBuffSize`)
- **Concurrent Workers**: More workers = better throughput and resilience
- **Caching**: First access to a document requires API calls; subsequent accesses use cache

## Limitations

- **Channel-Only**: Currently supports documents from Telegram channels only
- **Single Channel**: Each worker pool is bound to a single channel ID
- **Bot Tokens Required**: Requires valid bot tokens (not user accounts)
- **4KB Alignment**: Telegram requires 4KB-aligned offsets (handled automatically)

## Example: Complete HTTP Streaming Server

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "strconv"

    "github.com/amirdaaee/TGMon/internal/stream"
    "github.com/amirdaaee/TGMon/internal/tlg"
)

func main() {
    // Setup
    sessCfg := &tlg.SessionConfig{
        AppID:      12345678,
        AppHash:    "your-hash",
        SessionDir: "./sessions",
    }

    tokens := []string{"token1", "token2"}
    channelID := int64(-1001234567890)

    pool, err := stream.NewWorkerPool(tokens, sessCfg, channelID, "./cache")
    if err != nil {
        log.Fatal(err)
    }

    // HTTP handler
    http.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
        msgIDStr := r.URL.Path[len("/stream/"):]
        msgID, _ := strconv.Atoi(msgIDStr)

        // Get document size (simplified - in production, fetch from DB)
        worker := pool.GetNextWorker()
        doc, err := worker.GetDoc(r.Context(), msgID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }

        fileSize := doc.GetSize()
        offset := int64(0)
        end := fileSize - 1

        // TODO: Parse Range header for partial content

        streamer, err := pool.Stream(r.Context(), msgID, offset, end)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", doc.GetMimeType())
        w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

        io.Copy(w, streamer.GetBuffer())
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Troubleshooting

### Workers Fail to Initialize
- Check bot tokens are valid
- Verify App ID and App Hash are correct
- Ensure session directory is writable
- Check network connectivity and proxy settings

### Streaming Fails with Flood Wait
- This is handled automatically by the pool
- Add more workers to distribute load
- Check if you're hitting Telegram rate limits

### Cache Issues
- Ensure cache directory is writable
- Clear cache directory if you suspect corruption
- Check disk space availability

## See Also

- `internal/tlg`: Telegram client implementation
- `internal/web/stream.go`: Example HTTP streaming handler
- `internal/stream/downloader`: Low-level download implementation
