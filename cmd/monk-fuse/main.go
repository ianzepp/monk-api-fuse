package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/ianzepp/monk-api-fuse/pkg/monkapi"
	"github.com/ianzepp/monk-api-fuse/pkg/monkfs"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "mount":
		mountCmd()
	case "unmount":
		unmountCmd()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func mountCmd() {
	mountFlags := flag.NewFlagSet("mount", flag.ExitOnError)
	apiURL := mountFlags.String("api-url", "http://localhost:8000", "Monk API base URL")
	token := mountFlags.String("token", "", "JWT authentication token")
	debug := mountFlags.Bool("debug", false, "Enable FUSE debug logging")

	mountFlags.Parse(os.Args[2:])

	if mountFlags.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: monk-fuse mount [options] MOUNTPOINT")
		mountFlags.PrintDefaults()
		os.Exit(1)
	}

	mountPoint := mountFlags.Arg(0)

	// Get token from environment if not provided
	if *token == "" {
		*token = os.Getenv("MONK_TOKEN")
	}
	if *token == "" {
		log.Fatal("Error: No token provided. Use --token or set MONK_TOKEN environment variable")
	}

	// Create API client
	apiClient := monkapi.NewClient(*apiURL, *token)

	// Create FUSE filesystem
	root := monkfs.NewMonkFS(apiClient)

	// Mount options
	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			Name:          "monk-fuse",
			FsName:        "monk",
			Debug:         *debug,
			AllowOther:    false,
			DisableXAttrs: true,
		},
	}

	// Mount the filesystem
	server, err := fs.Mount(mountPoint, root, opts)
	if err != nil {
		log.Fatalf("Mount failed: %v", err)
	}

	fmt.Printf("Mounted Monk File API at: %s\n", mountPoint)
	fmt.Printf("API URL: %s\n", *apiURL)
	fmt.Println("Press Ctrl+C to unmount...")

	// Handle signals for graceful unmount
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nUnmounting...")
		err := server.Unmount()
		if err != nil {
			log.Printf("Unmount error: %v", err)
		}
	}()

	// Wait for filesystem to be unmounted
	server.Wait()
	fmt.Println("Unmounted successfully")
}

func unmountCmd() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: monk-fuse unmount MOUNTPOINT")
		os.Exit(1)
	}

	mountPoint := os.Args[2]

	// Use umount command (works on macOS)
	cmd := exec.Command("umount", mountPoint)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Unmount failed: %v", err)
	}

	fmt.Printf("Unmounted: %s\n", mountPoint)
}

func printUsage() {
	fmt.Println("monk-fuse - Mount Monk File API as a local filesystem")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  monk-fuse mount [options] MOUNTPOINT")
	fmt.Println("  monk-fuse unmount MOUNTPOINT")
	fmt.Println("  monk-fuse help")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  mount      Mount the filesystem")
	fmt.Println("  unmount    Unmount the filesystem")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Mount options:")
	fmt.Println("  --api-url URL     Monk API base URL (default: http://localhost:8000)")
	fmt.Println("  --token TOKEN     JWT authentication token (or set MONK_TOKEN env var)")
	fmt.Println("  --debug           Enable FUSE debug logging")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Mount with token from environment")
	fmt.Println("  export MONK_TOKEN=$(monk auth token)")
	fmt.Println("  monk-fuse mount ~/monk-data")
	fmt.Println()
	fmt.Println("  # Mount with explicit token")
	fmt.Println("  monk-fuse mount --token eyJhbGc... ~/monk-data")
	fmt.Println()
	fmt.Println("  # Unmount")
	fmt.Println("  monk-fuse unmount ~/monk-data")
}
