(() => {
	const $ = (sel) => document.querySelector(sel);
	const $$ = (sel) => document.querySelectorAll(sel);

	// ── i18n ────────────────────────────────────────────────

	const i18n = {
		zh: {
			subtitle: "跨设备内容分享",
			tokenLabel: "Token (可选)",
			save: "保存",
			checkConn: "检查连接",
			push: "推送",
			pull: "拉取",
			text: "文本",
			file: "文件",
			pushPlaceholder: "输入要分享的文本内容…",
			dropHint: "点击选择文件或拖拽到此处",
			defaultPlaceholder: "默认",
			seconds: "秒",
			copy: "复制",
			enterKey: "输入 Key",
			deleteAfterPull: "拉取后删除",
			textContent: "文本内容",
			download: "下载",
			delete: "删除",
			settings: "设置",
			toggleTheme: "切换主题",
			// dynamic strings
			settingsSaved: "设置已保存",
			checking: "检查中…",
			noConnect: "✗ 无法连接",
			enterText: "请输入文本内容",
			selectFile: "请选择文件",
			pushing: "推送中…",
			pushOk: "推送成功",
			pushFail: "推送失败",
			pullFail: "拉取失败",
			pulling: "拉取中…",
			pullOk: "拉取成功",
			enterKeyWarn: "请输入 Key",
			deleteFail: "删除失败",
			deleteOk: "删除成功",
			deleted: "已删除",
			keyCopied: "Key 已复制",
			contentCopied: "内容已复制",
			reqFail: "请求失败",
			metaType: "类型",
			metaSize: "大小",
			metaFile: "文件",
		},
		en: {
			subtitle: "Cross-Device Content Sharing",
			tokenLabel: "Token (optional)",
			save: "Save",
			checkConn: "Test Connection",
			push: "Push",
			pull: "Pull",
			text: "Text",
			file: "File",
			pushPlaceholder: "Enter text to share…",
			dropHint: "Click to select file or drag & drop here",
			defaultPlaceholder: "Default",
			seconds: "sec",
			copy: "Copy",
			enterKey: "Enter Key",
			deleteAfterPull: "Delete after pull",
			textContent: "Text Content",
			download: "Download",
			delete: "Delete",
			settings: "Settings",
			toggleTheme: "Toggle Theme",
			// dynamic strings
			settingsSaved: "Settings saved",
			checking: "Checking…",
			noConnect: "✗ Cannot connect",
			enterText: "Please enter text",
			selectFile: "Please select a file",
			pushing: "Pushing…",
			pushOk: "Push successful",
			pushFail: "Push failed",
			pullFail: "Pull failed",
			pulling: "Pulling…",
			pullOk: "Pull successful",
			enterKeyWarn: "Please enter Key",
			deleteFail: "Delete failed",
			deleteOk: "Deleted successfully",
			deleted: "Deleted",
			keyCopied: "Key copied",
			contentCopied: "Content copied",
			reqFail: "Request failed",
			metaType: "Type",
			metaSize: "Size",
			metaFile: "File",
		},
	};

	let currentLang = localStorage.getItem("cs_lang") || "zh";

	function t(key) {
		return (i18n[currentLang] && i18n[currentLang][key]) || key;
	}

	function applyI18n() {
		$$("[data-i18n]").forEach((el) => {
			const key = el.getAttribute("data-i18n");
			if (i18n[currentLang][key] !== undefined) {
				el.textContent = i18n[currentLang][key];
			}
		});
		$$("[data-i18n-placeholder]").forEach((el) => {
			const key = el.getAttribute("data-i18n-placeholder");
			if (i18n[currentLang][key] !== undefined) {
				el.placeholder = i18n[currentLang][key];
			}
		});
		$$("[data-i18n-title]").forEach((el) => {
			const key = el.getAttribute("data-i18n-title");
			if (i18n[currentLang][key] !== undefined) {
				el.title = i18n[currentLang][key];
			}
		});
		// Update html lang attribute
		document.documentElement.lang = currentLang === "zh" ? "zh-CN" : "en";
	}

	function toggleLang() {
		currentLang = currentLang === "zh" ? "en" : "zh";
		localStorage.setItem("cs_lang", currentLang);
		applyI18n();
		const btn = $("#langToggle");
		btn.textContent = currentLang === "zh" ? "EN" : "中";
		btn.title = currentLang === "zh" ? "Switch to English" : "切换到中文";
	}

	// ── Theme ──────────────────────────────────────────────

	let currentTheme = localStorage.getItem("cs_theme") || "dark";

	function applyTheme(theme) {
		currentTheme = theme;
		document.documentElement.setAttribute("data-theme", theme);
		localStorage.setItem("cs_theme", theme);
		const btn = $("#themeToggle");
		btn.textContent = theme === "dark" ? "☀" : "☾";
		const metaColor = $("#metaThemeColor");
		if (metaColor) {
			metaColor.content = theme === "dark" ? "#0f1117" : "#f5f5f7";
		}
	}

	function toggleTheme() {
		applyTheme(currentTheme === "dark" ? "light" : "dark");
	}

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
		toast(t("settingsSaved"), "success");
	}

	function apiUrl(path) {
		return loadSettings().serverUrl + "/api/v1" + path;
	}

	function authHeaders() {
		const tk = loadSettings().token;
		return tk ? { Authorization: "Bearer " + tk } : {};
	}

	// ── Init ──────────────────────────────────────────────────

	// Apply saved theme
	applyTheme(currentTheme);

	// Apply saved language
	applyI18n();
	const langBtn = $("#langToggle");
	langBtn.textContent = currentLang === "zh" ? "EN" : "中";
	langBtn.title = currentLang === "zh" ? "Switch to English" : "切换到中文";

	// Init settings panel
	const s = loadSettings();
	$("#serverUrl").value = s.serverUrl;
	$("#authToken").value = s.token;

	// ── Event: lang toggle ────────────────────────────────────

	$("#langToggle").addEventListener("click", toggleLang);

	// ── Event: theme toggle ───────────────────────────────────

	$("#themeToggle").addEventListener("click", toggleTheme);

	// ── Event: settings toggle ────────────────────────────────

	$("#settingsToggle").addEventListener("click", () => {
		$("#settingsPanel").classList.toggle("hidden");
	});

	$("#saveSettings").addEventListener("click", saveSettings);

	$("#healthCheck").addEventListener("click", async () => {
		const el = $("#healthStatus");
		el.textContent = t("checking");
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
			el.textContent = t("noConnect");
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
		btn.textContent = t("pushing");

		try {
			let resp;
			if (isText) {
				const text = $("#pushText").value;
				if (!text.trim()) {
					toast(t("enterText"), "error");
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
					toast(t("selectFile"), "error");
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
				toast(`${t("pushFail")}: ${data.msg}`, "error");
				return;
			}

			const r = data.data;
			$("#resultKey").textContent = r.key;
			let meta = `${t("metaType")}: ${r.type} · ${t("metaSize")}: ${humanSize(r.size)} · TTL: ${humanDuration(r.ttl)}`;
			if (r.filename) meta += ` · ${t("metaFile")}: ${r.filename}`;
			$("#resultMeta").textContent = meta;
			$("#pushResult").classList.remove("hidden");

			toast(t("pushOk"), "success");
		} catch (e) {
			toast(t("reqFail") + ": " + e.message, "error");
		} finally {
			btn.disabled = false;
			btn.textContent = t("push");
		}
	});

	$("#copyKey").addEventListener("click", () => {
		copyText($("#resultKey").textContent);
		toast(t("keyCopied"), "success");
	});

	// ── Pull ──────────────────────────────────────────────────

	$("#pullBtn").addEventListener("click", async () => {
		const key = $("#pullKey").value.trim();
		if (!key) {
			toast(t("enterKeyWarn"), "error");
			return;
		}

		const btn = $("#pullBtn");
		btn.disabled = true;
		btn.textContent = t("pulling");

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
					toast(`${t("pullFail")}: ${data.msg}`, "error");
					return;
				}

				const r = data.data;
				if (r.text !== undefined) {
					$("#pullTextContent").textContent = r.text;
					$("#pullTextResult").classList.remove("hidden");
					$("#pullFileResult").classList.add("hidden");
					let meta = `Key: ${r.key} · ${t("metaSize")}: ${humanSize(r.size)} · ${t("metaType")}: ${r.content_type}`;
					if (r.deleted) meta += ` · ${t("deleted")}`;
					$("#pullMeta").textContent = meta;
					$("#pullResult").classList.remove("hidden");
					toast(t("pullOk"), "success");
					return;
				}
			}

			const streamResp = await fetch(apiUrl("/pull/" + key), { headers });
			if (!streamResp.ok) {
				const errData = await streamResp.json().catch(() => null);
				toast(
					`${t("pullFail")}: ${errData?.msg || "HTTP " + streamResp.status}`,
					"error",
				);
				return;
			}

			const blob = await streamResp.blob();
			const shareType = streamResp.headers.get("Crossshare-Type");
			const filename =
				streamResp.headers.get("Crossshare-Filename") || "download";
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
				link.textContent = t("download");
				$("#pullFileResult").classList.remove("hidden");
				$("#pullTextResult").classList.add("hidden");
			}

			let meta = `Key: ${key} · ${t("metaSize")}: ${humanSize(blob.size)}`;
			if (deleted) meta += ` · ${t("deleted")}`;
			$("#pullMeta").textContent = meta;
			$("#pullResult").classList.remove("hidden");
			toast(t("pullOk"), "success");
		} catch (e) {
			toast(t("reqFail") + ": " + e.message, "error");
		} finally {
			btn.disabled = false;
			btn.textContent = t("pull");
		}
	});

	$("#copyText").addEventListener("click", () => {
		copyText($("#pullTextContent").textContent);
		toast(t("contentCopied"), "success");
	});

	// ── Delete ────────────────────────────────────────────────

	$("#deleteBtn").addEventListener("click", async () => {
		const key = $("#deleteKey").value.trim();
		if (!key) {
			toast(t("enterKeyWarn"), "error");
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
				toast(`${t("deleteFail")}: ${data.msg}`, "error");
				return;
			}

			$("#deleteMeta").textContent = `Key: ${data.data.key} · ${t("deleted")}`;
			$("#deleteResult").classList.remove("hidden");
			toast(t("deleteOk"), "success");
		} catch (e) {
			toast(t("reqFail") + ": " + e.message, "error");
		} finally {
			btn.disabled = false;
		}
	});

	// ── Helpers ───────────────────────────────────────────────

	function copyText(text) {
		if (navigator.clipboard && window.isSecureContext) {
			navigator.clipboard.writeText(text).catch(() => copyFallback(text));
			return;
		}
		copyFallback(text);
	}

	function copyFallback(text) {
		const ta = document.createElement("textarea");
		ta.value = text;
		ta.setAttribute("readonly", "");
		ta.style.cssText =
			"position:fixed;left:-9999px;top:0;opacity:0;font-size:16px";
		document.body.appendChild(ta);

		const isiOS = /ipad|iphone|ipod/i.test(navigator.userAgent);
		if (isiOS) {
			const range = document.createRange();
			range.selectNodeContents(ta);
			const sel = window.getSelection();
			sel.removeAllRanges();
			sel.addRange(range);
			ta.setSelectionRange(0, text.length);
		} else {
			ta.select();
		}

		try {
			document.execCommand("copy");
		} catch {
			// ignore
		}
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

	// ── Select all on first focus ─────────────────────────────
	document
		.querySelectorAll(
			'input[type="text"], input[type="number"], input[type="password"], textarea',
		)
		.forEach((el) => {
			el.addEventListener("focus", () => {
				requestAnimationFrame(() => el.select());
			});
		});
})();
