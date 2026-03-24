(() => {
  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => document.querySelectorAll(sel);

  // ── Settings ──────────────────────────────────────────────

  const defaults = {
    serverUrl: window.__CROSSSHARE_SERVER__ || window.location.origin,
    token: "",
  };

  function loadSettings() {
    return {
      serverUrl: localStorage.getItem("cs_server") || defaults.serverUrl,
      token: localStorage.getItem("cs_token") || defaults.token,
    };
  }

  function saveSettings() {
    const url = $("#serverUrl").value.replace(/\/+$/, "");
    localStorage.setItem("cs_server", url);
    localStorage.setItem("cs_token", $("#authToken").value);
    toast("设置已保存", "success");
  }

  function apiUrl(path) {
    return loadSettings().serverUrl + "/api/v1" + path;
  }

  function authHeaders() {
    const t = loadSettings().token;
    return t ? { Authorization: "Bearer " + t } : {};
  }

  // ── Init settings panel ───────────────────────────────────

  const s = loadSettings();
  $("#serverUrl").value = s.serverUrl;
  $("#authToken").value = s.token;

  $("#settingsToggle").addEventListener("click", () => {
    $("#settingsPanel").classList.toggle("hidden");
  });

  $("#saveSettings").addEventListener("click", saveSettings);

  $("#healthCheck").addEventListener("click", async () => {
    const el = $("#healthStatus");
    el.textContent = "检查中…";
    el.style.color = "var(--text2)";
    try {
      const resp = await fetch(apiUrl("/health"), {
        headers: authHeaders(),
      });
      const data = await resp.json();
      if (data.code === 0) {
        el.textContent = "✓ " + data.data.status;
        el.style.color = "var(--success)";
      } else {
        el.textContent = "✗ " + data.msg;
        el.style.color = "var(--danger)";
      }
    } catch {
      el.textContent = "✗ 无法连接";
      el.style.color = "var(--danger)";
    }
  });

  // ── Tabs ──────────────────────────────────────────────────

  $$(".tab").forEach((tab) => {
    tab.addEventListener("click", () => {
      $$(".tab").forEach((t) => t.classList.remove("active"));
      $$(".tab-content").forEach((c) => c.classList.remove("active"));
      tab.classList.add("active");
      $(`#${tab.dataset.tab}Tab`).classList.add("active");
    });
  });

  // ── Push: mode switch ─────────────────────────────────────

  $$(".mode").forEach((btn) => {
    btn.addEventListener("click", () => {
      $$(".mode").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      const isText = btn.dataset.mode === "text";
      $("#textMode").classList.toggle("hidden", !isText);
      $("#fileMode").classList.toggle("hidden", isText);
    });
  });

  // ── Push: file handling ───────────────────────────────────

  let selectedFile = null;

  const dropZone = $("#dropZone");
  const fileInput = $("#fileInput");

  dropZone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropZone.classList.add("dragover");
  });

  dropZone.addEventListener("dragleave", () => {
    dropZone.classList.remove("dragover");
  });

  dropZone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropZone.classList.remove("dragover");
    if (e.dataTransfer.files.length) {
      selectFile(e.dataTransfer.files[0]);
    }
  });

  fileInput.addEventListener("change", () => {
    if (fileInput.files.length) {
      selectFile(fileInput.files[0]);
    }
  });

  function selectFile(file) {
    selectedFile = file;
    const info = $("#fileInfo");
    info.classList.remove("hidden");
    info.innerHTML = `<span>${file.name}</span><span>${humanSize(file.size)}</span>`;
  }

  // ── Push ──────────────────────────────────────────────────

  $("#pushBtn").addEventListener("click", async () => {
    const btn = $("#pushBtn");
    const isText = $(".mode.active").dataset.mode === "text";
    btn.disabled = true;
    btn.textContent = "推送中…";

    try {
      let resp;
      if (isText) {
        const text = $("#pushText").value;
        if (!text.trim()) {
          toast("请输入文本内容", "error");
          return;
        }
        const body = { text };
        const ttl = parseInt($("#pushTtl").value);
        if (ttl > 0) body.ttl = ttl;

        resp = await fetch(apiUrl("/push/text"), {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...authHeaders(),
          },
          body: JSON.stringify(body),
        });
      } else {
        if (!selectedFile) {
          toast("请选择文件", "error");
          return;
        }
        const headers = {
          "Content-Type": "application/octet-stream",
          Filename: selectedFile.name,
          ...authHeaders(),
        };
        const ttl = parseInt($("#pushTtl").value);
        if (ttl > 0) headers["X-TTL"] = String(ttl);

        resp = await fetch(apiUrl("/push/binary"), {
          method: "POST",
          headers,
          body: selectedFile,
        });
      }

      const data = await resp.json();
      if (data.code !== 0) {
        toast(`推送失败: ${data.msg}`, "error");
        return;
      }

      const r = data.data;
      $("#resultKey").textContent = r.key;
      let meta = `类型: ${r.type} · 大小: ${humanSize(r.size)} · TTL: ${humanDuration(r.ttl)}`;
      if (r.filename) meta += ` · 文件: ${r.filename}`;
      $("#resultMeta").textContent = meta;
      $("#pushResult").classList.remove("hidden");

      toast("推送成功", "success");
    } catch (e) {
      toast("请求失败: " + e.message, "error");
    } finally {
      btn.disabled = false;
      btn.textContent = "推送";
    }
  });

  $("#copyKey").addEventListener("click", () => {
    copyText($("#resultKey").textContent);
    toast("Key 已复制", "success");
  });

  // ── Pull ──────────────────────────────────────────────────

  $("#pullBtn").addEventListener("click", async () => {
    const key = $("#pullKey").value.trim();
    if (!key) {
      toast("请输入 Key", "error");
      return;
    }

    const btn = $("#pullBtn");
    btn.disabled = true;
    btn.textContent = "拉取中…";

    try {
      const headers = {
        ...authHeaders(),
      };
      if ($("#deleteAfterPull").checked) {
        headers["Delete-After-Pull"] = "true";
      }

      const jsonResp = await fetch(apiUrl("/pull/" + key), {
        headers: { ...headers, Accept: "application/json" },
      });

      const ct = jsonResp.headers.get("Content-Type") || "";

      if (ct.includes("application/json")) {
        const data = await jsonResp.json();
        if (data.code !== 0) {
          toast(`拉取失败: ${data.msg}`, "error");
          return;
        }

        const r = data.data;
        if (r.text !== undefined) {
          $("#pullTextContent").textContent = r.text;
          $("#pullTextResult").classList.remove("hidden");
          $("#pullFileResult").classList.add("hidden");
          let meta = `Key: ${r.key} · 大小: ${humanSize(r.size)} · 类型: ${r.content_type}`;
          if (r.deleted) meta += " · 已删除";
          $("#pullMeta").textContent = meta;
          $("#pullResult").classList.remove("hidden");
          toast("拉取成功", "success");
          return;
        }
      }

      const streamResp = await fetch(apiUrl("/pull/" + key), { headers });
      if (!streamResp.ok) {
        const errData = await streamResp.json().catch(() => null);
        toast(`拉取失败: ${errData?.msg || "HTTP " + streamResp.status}`, "error");
        return;
      }

      const blob = await streamResp.blob();
      const shareType = streamResp.headers.get("Crossshare-Type");
      const filename = streamResp.headers.get("Crossshare-Filename") || "download";
      const deleted = streamResp.headers.get("Key-Deleted") === "true";

      if (shareType === "Text") {
        const text = await blob.text();
        $("#pullTextContent").textContent = text;
        $("#pullTextResult").classList.remove("hidden");
        $("#pullFileResult").classList.add("hidden");
      } else {
        const url = URL.createObjectURL(blob);
        $("#pullFileName").textContent = filename;
        const link = $("#pullFileLink");
        link.href = url;
        link.download = filename;
        link.textContent = "下载";
        $("#pullFileResult").classList.remove("hidden");
        $("#pullTextResult").classList.add("hidden");
      }

      let meta = `Key: ${key} · 大小: ${humanSize(blob.size)}`;
      if (deleted) meta += " · 已删除";
      $("#pullMeta").textContent = meta;
      $("#pullResult").classList.remove("hidden");
      toast("拉取成功", "success");
    } catch (e) {
      toast("请求失败: " + e.message, "error");
    } finally {
      btn.disabled = false;
      btn.textContent = "拉取";
    }
  });

  $("#copyText").addEventListener("click", () => {
    copyText($("#pullTextContent").textContent);
    toast("内容已复制", "success");
  });

  // ── Delete ────────────────────────────────────────────────

  $("#deleteBtn").addEventListener("click", async () => {
    const key = $("#deleteKey").value.trim();
    if (!key) {
      toast("请输入 Key", "error");
      return;
    }

    const btn = $("#deleteBtn");
    btn.disabled = true;

    try {
      const resp = await fetch(apiUrl("/pull/" + key), {
        method: "DELETE",
        headers: authHeaders(),
      });
      const data = await resp.json();
      if (data.code !== 0) {
        toast(`删除失败: ${data.msg}`, "error");
        return;
      }

      $("#deleteMeta").textContent = `Key: ${data.data.key} · 已删除`;
      $("#deleteResult").classList.remove("hidden");
      toast("删除成功", "success");
    } catch (e) {
      toast("请求失败: " + e.message, "error");
    } finally {
      btn.disabled = false;
    }
  });

  // ── Helpers ───────────────────────────────────────────────

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).catch(() => copyFallback(text));
    } else {
      copyFallback(text);
    }
  }

  function copyFallback(text) {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.cssText = "position:fixed;opacity:0";
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
  }

  function humanSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    const units = ["KiB", "MiB", "GiB"];
    let i = -1;
    let b = bytes;
    do {
      b /= 1024;
      i++;
    } while (b >= 1024 && i < units.length - 1);
    return b.toFixed(1) + " " + units[i];
  }

  function humanDuration(sec) {
    if (sec < 60) return sec + "s";
    if (sec < 3600) return Math.floor(sec / 60) + "m" + (sec % 60) + "s";
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    return m ? h + "h" + m + "m" : h + "h";
  }

  let toastTimer;
  function toast(msg, type) {
    const el = $("#toast");
    el.textContent = msg;
    el.className = "toast " + (type || "");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.classList.add("hidden"), 2500);
  }

  // ── Keyboard shortcut ─────────────────────────────────────

  document.addEventListener("keydown", (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      const pushActive = $("#pushTab").classList.contains("active");
      if (pushActive) {
        $("#pushBtn").click();
      }
    }
  });

})();
