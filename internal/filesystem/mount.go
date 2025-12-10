package filesystem

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/filesystem/src"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// MountOptions configures the filesystem mount behavior.
// It provides options for controlling FUSE mount parameters such as
// access permissions and debugging.
type MountOptions struct {
	// AllowOther allows other users (including containers) to access the filesystem
	// Note: This requires /etc/fuse.conf to have "user_allow_other" enabled
	AllowOther bool
	// Debug enables FUSE debug logging
	Debug bool
}

// MountWithOptions mounts the filesystem with custom options
func MountWithOptions(mountPoint string, srcs []src.ISrc, dbContainer db.IDbContainer, opts *MountOptions) (*fuse.Server, error) {
	ll := log.GetLogger(log.FuseModule).WithField("func", "Mount")
	ll.Infof("Mounting filesystem at: %s", mountPoint)

	if opts == nil {
		opts = &MountOptions{}
	}

	// Check if FUSE device is available (required for mounting)
	// This helps diagnose issues in Docker containers
	if _, err := os.Stat("/dev/fuse"); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("FUSE device /dev/fuse not found - container may need --device /dev/fuse or --privileged flag")
		}
		ll.WithError(err).Warn("Could not stat /dev/fuse - mount may fail")
	}

	// ensure mount point exists with proper permissions
	// Use 0755 for normal, 0777 if allow-other is enabled (needed for container access)
	mountPerms := os.FileMode(0755)
	if opts.AllowOther {
		mountPerms = 0777
	}
	if err := os.MkdirAll(mountPoint, mountPerms); err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	// Check if mount point is already a mount point (to avoid conflicts)
	// This is a best-effort check - it may not catch all cases
	var stat syscall.Stat_t
	if err := syscall.Stat(mountPoint, &stat); err == nil {
		// If the mount point is on a different device than root, it might already be mounted
		var rootStat syscall.Stat_t
		if err := syscall.Stat("/", &rootStat); err == nil && stat.Dev != rootStat.Dev {
			ll.Warnf("Mount point %s appears to be on a different filesystem - ensure it's not already mounted", mountPoint)
		}
	}

	// Create root filesystem
	root := NewMediaFS(srcs, dbContainer)
	// Create FUSE server
	fuseOpts := &fs.Options{}
	fuseOpts.Debug = opts.Debug
	fuseOpts.AllowOther = opts.AllowOther
	// Set timeouts for better performance and stability
	// These help with container access by caching attributes and entries
	// Longer timeouts reduce database load when containers scan directories frequently
	attrTimeout := 5 * time.Second
	entryTimeout := 5 * time.Second
	fuseOpts.AttrTimeout = &attrTimeout
	fuseOpts.EntryTimeout = &entryTimeout
	// NegativeTimeout of 0 means don't cache failed lookups (safer)
	zeroTimeout := time.Duration(0)
	fuseOpts.NegativeTimeout = &zeroTimeout
	// Set MaxBackground to handle concurrent requests from containers
	// Higher value is needed for large directory scans (1000+ files)
	// This allows more concurrent FUSE operations without blocking
	// Default is typically 12, increasing to 128 helps with large directories and concurrent access
	fuseOpts.MaxBackground = 128

	if opts.AllowOther {
		ll.Info("AllowOther enabled - filesystem will be accessible to other users/containers")
	}

	server, err := fs.Mount(mountPoint, root, fuseOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to mount filesystem: %w", err)
	}

	// After successful mount, try to set permissions for allow-other access
	if opts.AllowOther {
		if err := os.Chmod(mountPoint, 0777); err != nil {
			ll.WithError(err).Warn("Failed to set mount point permissions to 0777 (this may be normal)")
		}
	}

	ll.Info("Filesystem mounted successfully")
	return server, nil
}

// Unmount unmounts the filesystem
func Unmount(mountPoint string) error {
	ll := log.GetLogger(log.FuseModule).WithField("func", "Unmount")
	ll.Infof("Unmounting filesystem at: %s", mountPoint)

	// Try to unmount using fusermount
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		// If that fails, try with MNT_FORCE
		ll.WithError(err).Error("failed to unmount filesystem using fusermount. trying with MNT_FORCE")
		if err := syscall.Unmount(mountPoint, syscall.MNT_FORCE); err != nil {
			return fmt.Errorf("failed to unmount: %w", err)
		}
	}

	ll.Info("Filesystem unmounted successfully")
	return nil
}
