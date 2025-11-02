package purgo

import (
	"bytes"
	"fmt"
	"io"

	"github.com/boris-chu/go-openzl/internal/codec"
	"github.com/boris-chu/go-openzl/internal/frame"
	"github.com/boris-chu/go-openzl/internal/graph"
)

// Reader provides streaming decompression using the io.Reader interface.
//
// This allows incremental decompression without loading all data into memory.
// The Reader decompresses an OpenZL frame on the first Read() call and buffers
// the output for subsequent reads.
//
// Example usage:
//
//	file, _ := os.Open("data.zl")
//	reader, _ := purgo.NewReader(file)
//	defer reader.Close()
//
//	buffer := make([]byte, 4096)
//	for {
//	    n, err := reader.Read(buffer)
//	    if err == io.EOF {
//	        break
//	    }
//	    // Process buffer[:n]
//	}
type Reader struct {
	// source is the underlying compressed data reader
	source io.Reader

	// buffer holds decompressed data waiting to be read
	buffer *bytes.Buffer

	// initialized tracks whether the frame has been parsed
	initialized bool

	// eof tracks whether we've reached end of decompressed data
	eof bool

	// err stores any error that occurred during initialization
	err error
}

// NewReader creates a new streaming decompression reader.
//
// The frame is not parsed immediately - parsing happens on the first Read() call.
// This allows for lazy initialization and better error handling.
//
// Parameters:
//   - r: io.Reader containing OpenZL compressed data
//
// Returns:
//   - *Reader: Streaming decompression reader
//   - error: Always nil (errors are deferred to first Read)
//
// Example:
//
//	file, _ := os.Open("data.zl")
//	reader, _ := purgo.NewReader(file)
//	defer reader.Close()
//
//	io.Copy(os.Stdout, reader) // Decompress to stdout
func NewReader(r io.Reader) (*Reader, error) {
	return &Reader{
		source:      r,
		buffer:      new(bytes.Buffer),
		initialized: false,
		eof:         false,
	}, nil
}

// Read reads decompressed data into p.
//
// On the first call, Read parses the OpenZL frame, executes the compression
// graph, and buffers all decompressed output. Subsequent calls serve data
// from the buffer.
//
// This implements the io.Reader interface.
//
// Parameters:
//   - p: Destination buffer for decompressed data
//
// Returns:
//   - n: Number of bytes read into p
//   - err: io.EOF when all data has been read, or other error
//
// Example:
//
//	reader, _ := purgo.NewReader(compressedData)
//	buffer := make([]byte, 1024)
//	for {
//	    n, err := reader.Read(buffer)
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    process(buffer[:n])
//	}
func (r *Reader) Read(p []byte) (n int, err error) {
	// Initialize on first read
	if !r.initialized {
		if err := r.initialize(); err != nil {
			r.err = err
			return 0, err
		}
		r.initialized = true
	}

	// If initialization failed, return the error
	if r.err != nil {
		return 0, r.err
	}

	// If we've already returned EOF, keep returning EOF
	if r.eof {
		return 0, io.EOF
	}

	// Read from buffer
	n, err = r.buffer.Read(p)
	if err == io.EOF {
		// Mark EOF so subsequent reads return EOF immediately
		r.eof = true
	}
	return n, err
}

// initialize parses the frame and decompresses all data into the buffer.
//
// This is called automatically on the first Read() call.
func (r *Reader) initialize() error {
	// Read all compressed data from source
	// Note: OpenZL frames are typically not huge, and we need the full frame
	// to parse the header. For truly streaming decompression (large files),
	// OpenZL would need multi-frame support.
	compressed, err := io.ReadAll(r.source)
	if err != nil {
		return fmt.Errorf("purgo: read compressed data failed: %w", err)
	}

	if len(compressed) == 0 {
		return fmt.Errorf("purgo: empty input")
	}

	// Step 1: Parse OpenZL frame
	frameReader := frame.NewReader(bytes.NewReader(compressed))
	f, err := frameReader.ReadFrame()
	if err != nil {
		return fmt.Errorf("purgo: parse frame failed: %w", err)
	}

	// Verify we have exactly one output
	if len(f.Outputs) != 1 {
		return fmt.Errorf("purgo: expected 1 output, got %d", len(f.Outputs))
	}

	// Step 2: Parse compression graph
	parser := graph.NewParser(f.Payload)
	g, graphSize, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("purgo: parse graph failed: %w", err)
	}

	// Step 3: Execute compression graph to decompress
	executor := graph.NewExecutor(codec.DefaultRegistry())
	compressedData := f.Payload[graphSize:]
	outputSizes := []uint64{f.Outputs[0].DecompressedSize}

	outputs, err := executor.Execute(g, compressedData, outputSizes)
	if err != nil {
		return fmt.Errorf("purgo: execute graph failed: %w", err)
	}

	// Step 4: Write decompressed data to buffer
	if _, err := r.buffer.Write(outputs[0]); err != nil {
		return fmt.Errorf("purgo: buffer write failed: %w", err)
	}

	return nil
}

// Close closes the reader and releases resources.
//
// If the underlying source implements io.Closer, it will be closed.
//
// Returns:
//   - error: Error from closing underlying source, if any
//
// Example:
//
//	reader, _ := purgo.NewReader(file)
//	defer reader.Close() // Always close to release resources
func (r *Reader) Close() error {
	if closer, ok := r.source.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
