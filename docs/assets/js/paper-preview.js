// paper-preview.js — renders the PDF previews on the docs site.
//
// Two fenced block languages are handled:
//
//     ```pdf              a path to a committed PDF, e.g. assets/pdf/showcase.pdf
//     ```pdf-example      the name of a registered example, e.g. imagegrid
//
// The second kind is generated in the browser by paper.wasm from the same
// GetPaper builder the page shows as its code sample, so the preview cannot
// drift from the code beside it. The wasm is ~4.7MB gzipped, so it is fetched
// lazily: a page with no pdf-example fence never requests it, and the
// instantiation is cached for the whole SPA session rather than per route.
//
// This replaces docsify-pdf-embed-plugin, which round-tripped pending embeds
// through localStorage, required $docsify.executeScript, and built absolute
// URLs from location.hostname only — dropping the /paper/ project-pages prefix.
(function () {
  "use strict";

  var WASM_DIR = "assets/wasm/";
  var VIEWER_HEIGHT = "50rem";

  var wasmReady = null; // cached instantiation promise, shared across routes
  var assetCache = new Map(); // repo-relative path -> Uint8Array
  var pending = []; // previews awaiting the current render pass
  var liveURLs = []; // object URLs to revoke when the next route renders
  var seq = 0;

  // Resolve against the document's own URL so the site works under a
  // project-pages path prefix such as /paper/. document.baseURI is absolute,
  // which URL() requires as a base; the route hash does not affect resolution.
  function siteURL(rel) {
    return new URL(rel, document.baseURI).href;
  }

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text) node.textContent = text;
    return node;
  }

  function loadScript(src) {
    return new Promise(function (resolve, reject) {
      var s = document.createElement("script");
      s.src = src;
      s.onload = resolve;
      s.onerror = function () {
        reject(new Error("failed to load " + src));
      };
      document.head.appendChild(s);
    });
  }

  // instantiateStreaming rejects unless the response Content-Type is exactly
  // application/wasm, which plain static file servers do not always send —
  // Python's http.server only learned the .wasm mapping in 3.11, and `make site`
  // uses it. Stream when we can, fall back to buffering when we cannot, so the
  // local flow works anywhere. The playground does the same for the same reason.
  async function instantiate(src, importObject) {
    var response = await fetch(src);
    if (!response.ok) {
      throw new Error("fetch " + src + ": HTTP " + response.status);
    }
    if (WebAssembly.instantiateStreaming) {
      try {
        return await WebAssembly.instantiateStreaming(
          response.clone(),
          importObject,
        );
      } catch (_) {
        // Fall through: almost always a Content-Type the browser refused.
      }
    }
    return WebAssembly.instantiate(await response.arrayBuffer(), importObject);
  }

  // paper-fs.js must be evaluated before wasm_exec.js: the latter only installs
  // its own stub filesystem when globalThis.fs is absent.
  function ensureWasm() {
    if (wasmReady) return wasmReady;

    wasmReady = (async function () {
      if (!globalThis.paperFS)
        await loadScript(siteURL("assets/js/paper-fs.js"));
      if (typeof Go === "undefined")
        await loadScript(siteURL(WASM_DIR + "wasm_exec.js"));

      var go = new Go();
      var src = siteURL(WASM_DIR + "paper.wasm");
      var result = await instantiate(src, go.importObject);
      // main() ends in select{}, so this never resolves; the exports stay
      // callable for the lifetime of the page.
      go.run(result.instance);

      for (
        var i = 0;
        i < 50 && typeof globalThis.paperGenerateExample !== "function";
        i++
      ) {
        await new Promise(function (r) {
          setTimeout(r, 20);
        });
      }
      if (typeof globalThis.paperGenerateExample !== "function") {
        throw new Error("paper.wasm loaded but registered no exports");
      }
    })();

    // A failed load must not be cached, or every later preview inherits it.
    wasmReady.catch(function () {
      wasmReady = null;
    });
    return wasmReady;
  }

  // The map key is the repo-relative path Go passes to os.ReadFile; the URL
  // drops the leading "docs/" because the deployed site root is docs/ itself.
  async function ensureAssets(paths) {
    for (const path of paths) {
      if (assetCache.has(path)) {
        globalThis.paperFS.set(path, assetCache.get(path));
        continue;
      }
      var url = siteURL(path.replace(/^docs\//, ""));
      var response = await fetch(url);
      if (!response.ok) {
        throw new Error("fetch " + url + ": HTTP " + response.status);
      }
      var bytes = new Uint8Array(await response.arrayBuffer());
      assetCache.set(path, bytes);
      globalThis.paperFS.set(path, bytes);
    }
  }

  function unwrap(result, what) {
    if (!result || result.error) {
      throw new Error(
        result && result.error ? result.error : what + " returned nothing",
      );
    }
    return result;
  }

  async function generate(name) {
    await ensureWasm();
    var assets =
      unwrap(globalThis.paperExampleAssets(name), "paperExampleAssets")
        .assets || [];
    await ensureAssets(assets);

    var base64 = unwrap(
      globalThis.paperGenerateExample(name),
      "paperGenerateExample",
    ).pdf;
    var binary = atob(base64);
    var bytes = new Uint8Array(binary.length);
    for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    return URL.createObjectURL(new Blob([bytes], { type: "application/pdf" }));
  }

  // A download link always accompanies the embed. iOS Safari and Android Chrome
  // render a blob <embed> as a blank box and raise no error, so there is nothing
  // to catch and fall back from — the link has to be there unconditionally.
  function showPDF(container, url, filename) {
    container.textContent = "";

    var frame = el("embed");
    frame.type = "application/pdf";
    frame.src = url;
    frame.style.width = "100%";
    frame.style.height = VIEWER_HEIGHT;
    frame.style.border = "0";
    container.appendChild(frame);

    var link = el("a", "paper-preview-download", "Download " + filename);
    link.href = url;
    link.setAttribute("download", filename);
    link.style.display = "inline-block";
    link.style.marginTop = ".5rem";
    container.appendChild(link);
  }

  function showError(container, name, message, retry) {
    container.textContent = "";
    var box = el("div", "paper-preview-error");
    box.style.border = "1px solid #c00";
    box.style.borderRadius = "4px";
    box.style.padding = "1rem";
    box.appendChild(
      el("strong", null, "Could not generate the " + name + " preview"),
    );
    var detail = el("pre", null, message);
    detail.style.whiteSpace = "pre-wrap";
    detail.style.margin = ".5rem 0";
    box.appendChild(detail);

    var button = el("button", null, "Retry");
    button.type = "button";
    button.addEventListener("click", retry);
    box.appendChild(button);

    container.appendChild(box);
  }

  function renderExample(container, name) {
    container.textContent = "";
    container.appendChild(
      el("p", "paper-preview-loading", "Generating " + name + " preview…"),
    );

    generate(name)
      .then(function (url) {
        liveURLs.push(url);
        showPDF(container, url, name + ".pdf");
      })
      .catch(function (err) {
        showError(
          container,
          name,
          String(err && err.message ? err.message : err),
          function () {
            renderExample(container, name);
          },
        );
      });
  }

  function renderStatic(container, path) {
    var url = siteURL(path);
    showPDF(container, url, path.split("/").pop());
  }

  // Docsify's markdown renderer runs before the HTML is in the DOM, so each
  // fence only leaves a placeholder here and is wired up in doneEach.
  function installRenderer() {
    var md = (window.$docsify.markdown = window.$docsify.markdown || {});
    var renderer = (md.renderer = md.renderer || {});
    var previous = renderer.code;

    renderer.code = function (code, lang) {
      var kind =
        lang === "pdf" ? "static" : lang === "pdf-example" ? "example" : null;
      if (!kind) {
        // Every other language falls through to whoever had the renderer before
        // us, or to docsify's own highlighter via this.origin.
        if (previous) return previous.apply(this, arguments);
        return this.origin.code.apply(this, arguments);
      }
      var id = "paper-preview-" + ++seq;
      pending.push({ id: id, kind: kind, value: String(code).trim() });
      return '<div class="paper-preview" id="' + id + '"></div>';
    };
  }

  window.$docsify = window.$docsify || {};
  installRenderer();

  window.$docsify.plugins = [
    function (hook) {
      hook.beforeEach(function (content, next) {
        // Docsify never unmounts a route, so blobs from the page we are
        // leaving are released here rather than in a teardown hook.
        liveURLs.splice(0).forEach(URL.revokeObjectURL);
        pending = [];
        next(content);
      });

      hook.doneEach(function () {
        pending.splice(0).forEach(function (item) {
          var container = document.getElementById(item.id);
          if (!container) return;
          if (item.kind === "static") {
            renderStatic(container, item.value);
          } else {
            renderExample(container, item.value);
          }
        });
      });
    },
  ].concat(window.$docsify.plugins || []);
})();
