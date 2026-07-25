// Tests for paper-fs.js, the in-memory filesystem Go's js/wasm runtime calls.
//
//     node --test docs/assets/js/
//
// These pin the contract that syscall/fs_js.go actually relies on. Getting any
// of it wrong fails in a way that is easy to miss: the renderer records an
// image-load issue, draws an error box, and still returns a valid PDF, so a
// "did a PDF come out?" check passes with the shim completely broken.
//
// No browser needed — the shim only touches globalThis, so plain node runs it.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test, beforeEach } from "node:test";

const SOURCE = new URL("./paper-fs.js", import.meta.url);

// paper-fs.js is a plain script (an IIFE assigning to globalThis), not a module,
// so it is evaluated rather than imported. Re-evaluating resets its state.
function loadShim() {
  delete globalThis.fs;
  delete globalThis.paperFS;
  new Function(readFileSync(SOURCE, "utf8"))();
}

// Promisified wrappers over the Node-style callbacks Go uses.
const call = (method, ...args) =>
  new Promise((resolve, reject) => {
    globalThis.fs[method](...args, (err, ...rest) =>
      err ? reject(err) : resolve(rest.length > 1 ? rest : rest[0]),
    );
  });

const open = (path) => call("open", path, 0, 0o444);
const close = (fd) => call("close", fd);
const fstat = (fd) => call("fstat", fd);
const read = (fd, buf, offset, length, position) =>
  call("read", fd, buf, offset, length, position);

beforeEach(loadShim);

test("open reports ENOENT for an unregistered path", async () => {
  await assert.rejects(open("missing.bin"), (err) => err.code === "ENOENT");
});

test("read and close reject a bad file descriptor", async () => {
  await assert.rejects(
    read(999, new Uint8Array(4), 0, 4, null),
    (err) => err.code === "EBADF",
  );
});

test("fstat exposes every field syscall.setStat reads", async () => {
  globalThis.paperFS.set("a.bin", new Uint8Array([1, 2, 3]));
  const fd = await open("a.bin");
  const stat = await fstat(fd);

  // setStat calls .Int() on each of these; .Int() on undefined panics and takes
  // the whole wasm instance with it.
  for (const field of [
    "dev", "ino", "mode", "nlink", "uid", "gid", "rdev", "size",
    "blksize", "blocks", "atimeMs", "mtimeMs", "ctimeMs",
  ]) {
    assert.equal(typeof stat[field], "number", `${field} must be a number`);
  }
  assert.equal(stat.size, 3);
  // syscall.Open always follows open with fstat + isDirectory(), and treats a
  // true result as a directory it should readdir.
  assert.equal(typeof stat.isDirectory, "function");
  assert.equal(stat.isDirectory(), false);
  await close(fd);
});

test("sequential reads advance the cursor and signal EOF with 0", async () => {
  const data = Uint8Array.from({ length: 1000 }, (_, i) => i % 251);
  globalThis.paperFS.set("big.bin", data);

  const fd = await open("big.bin");
  const out = new Uint8Array(data.length);
  const chunk = new Uint8Array(300);
  let total = 0;

  // position === null means "continue from this descriptor's cursor".
  for (;;) {
    const [n] = await read(fd, chunk, 0, chunk.length, null);
    if (n === 0) break; // os.ReadFile loops forever if EOF is never reported
    out.set(chunk.subarray(0, n), total);
    total += n;
    assert.ok(total <= data.length, "read past end of file");
  }

  assert.equal(total, data.length);
  assert.deepEqual(out, data);
  await close(fd);
});

test("a positioned read does not move the cursor", async () => {
  const data = Uint8Array.from({ length: 100 }, (_, i) => i);
  globalThis.paperFS.set("p.bin", data);
  const fd = await open("p.bin");

  const buf = new Uint8Array(10);
  const [n] = await read(fd, buf, 0, 10, 50); // syscall.Pread passes a number
  assert.equal(n, 10);
  assert.deepEqual(buf, data.subarray(50, 60));

  // The cursor must still be at 0, so the next cursor-read starts at the top.
  const [m] = await read(fd, buf, 0, 10, null);
  assert.equal(m, 10);
  assert.deepEqual(buf, data.subarray(0, 10));
  await close(fd);
});

test("an empty file reads as 0 bytes immediately", async () => {
  globalThis.paperFS.set("empty.bin", new Uint8Array([]));
  const fd = await open("empty.bin");
  assert.equal((await fstat(fd)).size, 0);
  const [n] = await read(fd, new Uint8Array(8), 0, 8, null);
  assert.equal(n, 0);
  await close(fd);
});

test("two descriptors on one file keep independent cursors", async () => {
  globalThis.paperFS.set("s.bin", Uint8Array.from({ length: 20 }, (_, i) => i));
  const a = await open("s.bin");
  const b = await open("s.bin");
  assert.notEqual(a, b);

  const buf = new Uint8Array(5);
  await read(a, buf, 0, 5, null); // advances a's cursor only
  const [n] = await read(b, buf, 0, 5, null);
  assert.equal(n, 5);
  assert.deepEqual(buf, Uint8Array.from({ length: 5 }, (_, i) => i));
  await close(a);
  await close(b);
});

test("readdir reports ENOTDIR rather than a generic failure", async () => {
  globalThis.paperFS.set("f.bin", new Uint8Array([0]));
  await assert.rejects(
    call("readdir", "f.bin"),
    (err) => err.code === "ENOTDIR",
  );
});

test("unsupported operations answer ENOSYS", async () => {
  await assert.rejects(call("unlink", "x"), (err) => err.code === "ENOSYS");
  await assert.rejects(call("mkdir", "d", 0o755), (err) => err.code === "ENOSYS");
});

test("stdout and stderr writes are line buffered to the console", () => {
  const lines = [];
  const realLog = console.log;
  console.log = (line) => lines.push(line);
  try {
    const enc = new TextEncoder();
    globalThis.fs.writeSync(1, enc.encode("hello "));
    assert.deepEqual(lines, [], "nothing flushes until a newline arrives");
    globalThis.fs.writeSync(1, enc.encode("world\n"));
    assert.deepEqual(lines, ["hello world"]);
  } finally {
    console.log = realLog;
  }
});

test("paperFS.set accepts an ArrayBuffer as well as a Uint8Array", async () => {
  globalThis.paperFS.set("ab.bin", new Uint8Array([9, 8, 7]).buffer);
  const fd = await open("ab.bin");
  assert.equal((await fstat(fd)).size, 3);
  await close(fd);
});
