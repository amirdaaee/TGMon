# TGMon

## Docker Setup with FUSE

This application uses FUSE (Filesystem in Userspace) to mount a virtual filesystem. When running in Docker, special configuration is required:

### Required Docker Flags

When running the container, you must include:

```bash
docker run \
  --cap-add SYS_ADMIN \
  --device /dev/fuse \
  --security-opt apparmor:unconfined \
  your-image-name
```

Or in docker-compose:

```yaml
services:
  tgmon:
    build: .
    cap_add:
      - SYS_ADMIN
    devices:
      - /dev/fuse:/dev/fuse
    security_opt:
      - apparmor:unconfined
```

### Important Notes

1. **Mount Point Location**: If your FUSE mount point (configured via `FUSE__MEDIA_DIR`) is inside a bind-mounted directory, ensure the mount point directory itself is created inside the container, not bind-mounted. For example:
   - ✅ Good: Bind mount `/host/data` to `/container/data`, then mount FUSE to `/container/data/fuse` (created inside container)
   - ❌ Problematic: Bind mount `/host/data/fuse` directly to `/container/data/fuse` and try to mount FUSE there

2. **Host Visibility**: To make the FUSE mount visible on the host, use `rshared` bind propagation:
   ```yaml
   volumes:
     - ./host/path:/container/path:rshared
   ```
   This allows mounts inside the container to propagate to the host. The host directory must be a mount point with shared propagation. If it's not, you may need to:
   ```bash
   # Make the host directory a shared mount point
   sudo mount --bind ./storage/tgmon-docker/tgmon-fuse ./storage/tgmon-docker/tgmon-fuse
   sudo mount --make-shared ./storage/tgmon-docker/tgmon-fuse
   ```

3. **Alternative**: Use a Docker volume for the mount point instead of a bind mount:
   ```yaml
   volumes:
     - fuse-mount:/tgmon-data/media
   ```

4. **Permissions**: The container user needs access to `/dev/fuse` and `SYS_ADMIN` capability to perform mounts.

### Troubleshooting

If the mount operation freezes:
- Verify `/dev/fuse` exists in the container: `docker exec <container> ls -l /dev/fuse`
- Check container logs for FUSE-related errors
- Ensure the mount point directory is not already a mount point
- Try using `--privileged` flag (less secure, but helps diagnose issues)
