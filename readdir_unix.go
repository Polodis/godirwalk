// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package godirwalk

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

const bufsize = 16 * 1024

func readdirents(osDirname string) (Dirents, error) {
	dh, err := os.Open(osDirname)
	if err != nil {
		return nil, errors.Wrap(err, "cannot Open")
	}

	var de *syscall.Dirent

	var entries Dirents
	fd := int(dh.Fd())

	scratchBuffer := make([]byte, bufsize)

	var nameBytes []byte                                     // will be updated to point to syscall.Dirent.Name
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&nameBytes)) // save slice header, so we can re-use each loop

	for {
		n, err := syscall.ReadDirent(fd, scratchBuffer)
		if err != nil {
			_ = dh.Close() // ignore potential error returned by Close
			return nil, errors.Wrap(err, "cannot ReadDirent")
		}
		if n <= 0 {
			break // end of directory reached
		}
		// Loop over the bytes returned by reading the directory entries.
		buf := scratchBuffer[:n]
		for len(buf) > 0 {
			// unshift left-most directory entry from the buffer
			de = (*syscall.Dirent)(unsafe.Pointer(&buf[0]))
			buf = buf[de.Reclen:]

			if de.Ino == 0 {
				continue // this item has been deleted, but not yet removed from directory
			}

			// Convert syscall.Dirent.Name, which is array of int8, to []byte,
			// by overwriting Cap, Len, and Data slice members to values from
			// syscall.Dirent.
			sh.Cap, sh.Len, sh.Data = len(de.Name), len(de.Name), uintptr(unsafe.Pointer(&de.Name[0]))
			nameLength := bytes.IndexByte(nameBytes, 0) // look for NULL byte
			if nameLength == -1 {
				nameLength = len(de.Name)
			}
			nameBytes = nameBytes[:nameLength]

			// Skip "." and ".." entries.
			if nameLength == 1 && nameBytes[0] == '.' || nameLength == 2 && nameBytes[0] == '.' && nameBytes[1] == '.' {
				continue
			}

			nameString := string(nameBytes)

			// Convert syscall constant, which is in purview of OS, to a
			// constant defined by Go, assumed by this project to be stable.
			var mode os.FileMode
			switch de.Type {
			case syscall.DT_REG:
				// regular file
			case syscall.DT_DIR:
				mode = os.ModeDir
			case syscall.DT_LNK:
				mode = os.ModeSymlink
			case syscall.DT_BLK:
				mode = os.ModeDevice
			case syscall.DT_CHR:
				mode = os.ModeDevice | os.ModeCharDevice
			case syscall.DT_FIFO:
				mode = os.ModeNamedPipe
			case syscall.DT_SOCK:
				mode = os.ModeSocket
			default:
				// If syscall returned unknown type (e.g., DT_UNKNOWN, DT_WHT),
				// then resolve actual mode by getting stat.
				fi, err := os.Stat(filepath.Join(osDirname, nameString))
				if err != nil {
					_ = dh.Close() // ignore potential error returned by Close
					return nil, errors.Wrap(err, "cannot Stat")
				}
				mode = fi.Mode()
			}

			// We only care about the bits that identify the type of a file
			// system node, and can ignore append, exclusive, temporary, setuid,
			// setgid, permission bits, and sticky bits, which are coincident to
			// bits which declare type of the file system node.
			entries = append(entries, &Dirent{name: nameString, modeType: mode & os.ModeType})
		}
	}
	if err = dh.Close(); err != nil {
		return nil, err
	}
	return entries, nil
}

func readdirnames(osDirname string) ([]string, error) {
	des, err := readdirents(osDirname)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(des))
	for i, v := range des {
		names[i] = v.name
	}
	return names, nil
}
