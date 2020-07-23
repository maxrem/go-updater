## Go updater

This will update your Go installation on Linux. Run with `sudo`.

#### Usage
```bash
sudo ./go-updater -version=1.14.6
```

To skip the download use `-skip-download` flag.

To use an alternate directory for the Go installation use `-directory` flag. Default directory is `/usr/local`.

If you only want to get the latest Go version printed to the console, use the `-check-version` flag.

---
Inspired by [Go download manager](https://github.com/usmanhalalit/go-download-manager)