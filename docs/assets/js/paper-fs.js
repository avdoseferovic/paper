// paper-fs.js — a read-only, in-memory filesystem for Go's js/wasm runtime.
//
// Go's `os.ReadFile` in a browser goes through `globalThis.fs`, which
// `wasm_exec.js` otherwise fills with stubs that answer ENOSYS to everything.
// That guard is `if (!globalThis.fs)`, so defining our own object *before*
// `wasm_exec.js` loads replaces the stub wholesale and lets the Paper library
// read images with no change to the library or to the documented example code.
//
// Load order matters:
//
//     <script src="paper-fs.js"></script>
//     <script src="wasm_exec.js"></script>
//
// Only `fs` is defined here. `wasm_exec.js` installs `globalThis.process` and
// `globalThis.path` behind their own separate guards and those stubs are fine —
// `syscall.Open` calls `path.resolve()`, which the stub implements, and nothing
// on the read path calls `process.cwd()`.
//
// Files are registered by the exact path Go will ask for:
//
//     paperFS.set("docs/assets/images/biplane.jpg", bytes);
//
// Paths are treated as opaque keys, not as a hierarchy — `syscall.Open` does not
// prepend the working directory, so whatever string the Go code passes to
// `os.ReadFile` is what arrives here.
(function () {
  "use strict";

  // Regular-file mode bits (S_IFREG). Without this, Go reads the entry as a
  // directory and follows a readdir path that has no meaning here.
  var S_IFREG = 0o100000;

  // Lowest fd we hand out. 0/1/2 stay reserved for stdin/stdout/stderr.
  var FIRST_FD = 3;

  var files = new Map(); // path -> Uint8Array
  var open = new Map(); // fd -> { path, bytes, cursor }
  var nextFd = FIRST_FD;

  // Line buffers for fd 1 and 2, so Go's log output arrives at the console in
  // whole lines rather than one call per write.
  var outputBuf = { 1: "", 2: "" };
  var decoder = new TextDecoder("utf-8");

  function err(code, message) {
    var e = new Error(message || code);
    e.code = code;
    return e;
  }

  function enosys() {
    return err("ENOSYS", "not implemented");
  }

  // Go's syscall.setStat reads every one of these fields with `.Int()`, which
  // panics on undefined. Returning a partial object kills the wasm instance on
  // the first stat, so all thirteen are always present.
  function statFor(bytes) {
    return {
      dev: 0,
      ino: 0,
      mode: S_IFREG | 0o444,
      nlink: 1,
      uid: 0,
      gid: 0,
      rdev: 0,
      size: bytes.length,
      blksize: 4096,
      blocks: Math.ceil(bytes.length / 512),
      atimeMs: 0,
      mtimeMs: 0,
      ctimeMs: 0,
      isDirectory: function () {
        return false;
      },
    };
  }

  function flush(fd) {
    var nl = outputBuf[fd].lastIndexOf("\n");
    if (nl === -1) {
      return;
    }
    var line = outputBuf[fd].substring(0, nl);
    outputBuf[fd] = outputBuf[fd].substring(nl + 1);
    if (fd === 2) {
      console.error(line);
    } else {
      console.log(line);
    }
  }

  globalThis.fs = {
    // The write flags keep wasm_exec.js's -1 sentinels. A read-only open
    // passes O_RDONLY (0), so none of these bits are ever set and the -1
    // values are never compared against anything meaningful.
    constants: {
      O_WRONLY: -1,
      O_RDWR: -1,
      O_CREAT: -1,
      O_TRUNC: -1,
      O_APPEND: -1,
      O_EXCL: -1,
      O_DIRECTORY: -1,
    },

    writeSync: function (fd, buf) {
      if (fd !== 1 && fd !== 2) {
        throw enosys();
      }
      outputBuf[fd] += decoder.decode(buf);
      flush(fd);
      return buf.length;
    },

    write: function (fd, buf, offset, length, position, callback) {
      if (offset !== 0 || length !== buf.length || position !== null) {
        callback(enosys());
        return;
      }
      var n = this.writeSync(fd, buf);
      callback(null, n);
    },

    open: function (path, flags, mode, callback) {
      var bytes = files.get(path);
      if (bytes === undefined) {
        callback(err("ENOENT", "no such file or directory: " + path));
        return;
      }
      var fd = nextFd++;
      open.set(fd, { path: path, bytes: bytes, cursor: 0 });
      callback(null, fd);
    },

    close: function (fd, callback) {
      open.delete(fd);
      callback(null);
    },

    // `syscall.Read` passes position === null, meaning "continue from this
    // descriptor's own cursor and advance it". `syscall.Pread` passes a
    // number, which reads from that offset and must leave the cursor alone.
    // Returning 0 is how EOF is signalled; without it os.ReadFile loops.
    read: function (fd, buffer, offset, length, position, callback) {
      var f = open.get(fd);
      if (f === undefined) {
        callback(err("EBADF", "bad file descriptor"));
        return;
      }

      var start =
        position === null || position === undefined ? f.cursor : position;
      if (start >= f.bytes.length) {
        callback(null, 0, buffer);
        return;
      }

      var end = Math.min(start + length, f.bytes.length);
      var n = end - start;
      buffer.set(f.bytes.subarray(start, end), offset);

      if (position === null || position === undefined) {
        f.cursor = end;
      }
      callback(null, n, buffer);
    },

    fstat: function (fd, callback) {
      var f = open.get(fd);
      if (f === undefined) {
        callback(err("EBADF", "bad file descriptor"));
        return;
      }
      callback(null, statFor(f.bytes));
    },

    stat: function (path, callback) {
      var bytes = files.get(path);
      if (bytes === undefined) {
        callback(err("ENOENT", "no such file or directory: " + path));
        return;
      }
      callback(null, statFor(bytes));
    },

    lstat: function (path, callback) {
      this.stat(path, callback);
    },

    // Reached only if a stat wrongly reported a directory. ENOTDIR makes that
    // mistake obvious instead of looking like a generic unsupported call.
    readdir: function (path, callback) {
      callback(err("ENOTDIR", "not a directory: " + path));
    },

    chmod: function (path, mode, callback) {
      callback(enosys());
    },
    chown: function (path, uid, gid, callback) {
      callback(enosys());
    },
    fchmod: function (fd, mode, callback) {
      callback(enosys());
    },
    fchown: function (fd, uid, gid, callback) {
      callback(enosys());
    },
    fsync: function (fd, callback) {
      callback(null);
    },
    ftruncate: function (fd, length, callback) {
      callback(enosys());
    },
    lchown: function (path, uid, gid, callback) {
      callback(enosys());
    },
    link: function (path, link, callback) {
      callback(enosys());
    },
    mkdir: function (path, perm, callback) {
      callback(enosys());
    },
    readlink: function (path, callback) {
      callback(enosys());
    },
    rename: function (from, to, callback) {
      callback(enosys());
    },
    rmdir: function (path, callback) {
      callback(enosys());
    },
    symlink: function (path, link, callback) {
      callback(enosys());
    },
    truncate: function (path, length, callback) {
      callback(enosys());
    },
    unlink: function (path, callback) {
      callback(enosys());
    },
    utimes: function (path, atime, mtime, callback) {
      callback(enosys());
    },
  };

  globalThis.paperFS = {
    set: function (path, bytes) {
      files.set(
        path,
        bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes),
      );
    },
    has: function (path) {
      return files.has(path);
    },
    clear: function () {
      files.clear();
    },
  };
})();
