(() => {
	const $ = (sel) => document.querySelector(sel);
	const $$ = (sel) => document.querySelectorAll(sel);

	// ── i18n ────────────────────────────────────────────────

	const i18n = {
		zh: {
			subtitle: "跨设备内容分享",
			serverUrlLabel: "服务器地址",
			tokenLabel: "Token (可选)",
			save: "保存",
			checkConn: "检查连接",
			push: "发送",
			pull: "接收",
			sendPanelSubtitle: "选择内容类型",
			receivePanelSubtitle: "输入 Key 或打开分享链接",
			openTextInput: "展开文本输入",
			selectedFiles: "已选文件",
			removeFile: "移除文件",
			text: "文本",
			file: "文件",
			pushPlaceholder: "输入要分享的文本内容…",
			collapsePaper: "收起信纸",
			dropHint: "拖拽文件到这里",
			dropSubHint: "或点击选择文件",
			dropReady: "文件已选择",
			dropReadyHint: "可继续发送",
			defaultPlaceholder: "默认",
			seconds: "秒",
			smartTransfer: "直连优先",
			serverTransfer: "服务器暂存",
			directTransferHint: "关闭后使用服务器暂存",
			copy: "复制",
			copyLink: "复制链接",
			showQr: "查看二维码",
			details: "详情",
			hideDetails: "收起详情",
			shareQr: "分享二维码",
			close: "关闭",
			cancel: "取消",
			saveToServer: "存到服务器",
			cleanup: "清理",
			moreActions: "更多操作",
			moreSettings: "更多设置",
			enterKey: "输入 Key",
			deleteAfterPull: "拉取后删除",
			textContent: "文本内容",
			download: "下载",
			clear: "清除",
			delete: "删除",
			settings: "设置",
			toggleTheme: "切换主题",
			// dynamic strings
			expired: "已过期",
			expiresIn: "过期倒计时",
			settingsSaved: "设置已保存",
			checking: "检查中…",
			noConnect: "✗ 无法连接",
			enterText: "请输入文本内容",
			selectFile: "请选择文件",
			pushing: "发送中…",
			pushOk: "发送成功",
			pushFail: "发送失败",
			p2pUnsupported: "当前浏览器不支持直连，已切换服务器传输",
			p2pWaiting: "等待接收方连接，请保持此页面打开",
			p2pSignalExchange: "交换连接信息",
			p2pLanCheck: "检测局域网直连",
			p2pInternetCheck: "尝试公网辅助连接",
			p2pSlow: "直连较慢，仍在尝试",
			p2pConnecting: "正在建立直连",
			p2pConnected: "直连已建立",
			p2pSending: "直连传输中",
			p2pSent: "直连传输完成",
			p2pNotStored: "直连传输完成，未存服务器",
			p2pReceiving: "正在接收直连内容",
			p2pDone: "直连接收完成",
			p2pFailed: "直连失败",
			p2pFallback: "正在改用服务器传输",
			p2pCancelled: "直连已取消",
			p2pLinkLabel: "直连连接",
			p2pSenderOffline: "发送方不在线或无法直连",
			p2pElapsed: "耗时",
			p2pRouteLabel: "连接",
			p2pRouteLan: "局域网直连",
			p2pRouteNat: "NAT 穿透",
			p2pRouteRelay: "TURN 中继",
			p2pCandidateLocal: "本地",
			p2pCandidateRemote: "对端",
			pullFail: "接收失败",
			pulling: "接收中…",
			pullOk: "接收成功",
			enterKeyWarn: "请输入 Key",
			deleteFail: "删除失败",
			deleteOk: "删除成功",
			deleted: "已删除",
			keyCopied: "Key 已复制",
			linkCopied: "链接已复制",
			contentCopied: "内容已复制",
			qrFail: "二维码生成失败",
			reqFail: "请求失败",
			metaType: "类型",
			metaSize: "大小",
			metaFile: "文件",
			metaFiles: "文件数",
			metaStored: "存储",
		},
		en: {
			subtitle: "Cross-Device Content Sharing",
			serverUrlLabel: "Server URL",
			tokenLabel: "Token (optional)",
			save: "Save",
			checkConn: "Test Connection",
			push: "Send",
			pull: "Receive",
			sendPanelSubtitle: "Choose content type",
			receivePanelSubtitle: "Enter a key or open a share link",
			openTextInput: "Open text input",
			selectedFiles: "Selected files",
			removeFile: "Remove file",
			text: "Text",
			file: "File",
			pushPlaceholder: "Enter text to share…",
			collapsePaper: "Collapse letter",
			dropHint: "Drop files here",
			dropSubHint: "or click to choose",
			dropReady: "File ready",
			dropReadyHint: "Ready to send",
			defaultPlaceholder: "Default",
			seconds: "sec",
			smartTransfer: "Direct First",
			serverTransfer: "Server Storage",
			directTransferHint: "Turn off to store on server",
			copy: "Copy",
			copyLink: "Copy Link",
			showQr: "QR Code",
			details: "Details",
			hideDetails: "Hide details",
			shareQr: "Share QR Code",
			close: "Close",
			cancel: "Cancel",
			saveToServer: "Save to Server",
			cleanup: "Clean",
			moreActions: "More actions",
			moreSettings: "More settings",
			enterKey: "Enter Key",
			deleteAfterPull: "Delete after pull",
			textContent: "Text Content",
			download: "Download",
			clear: "Clear",
			delete: "Delete",
			settings: "Settings",
			toggleTheme: "Toggle Theme",
			// dynamic strings
			expired: "Expired",
			expiresIn: "Expires in",
			settingsSaved: "Settings saved",
			checking: "Checking…",
			noConnect: "✗ Cannot connect",
			enterText: "Please enter text",
			selectFile: "Please select a file",
			pushing: "Sending…",
			pushOk: "Sent successfully",
			pushFail: "Send failed",
			p2pUnsupported: "Direct transfer is not supported, using server transfer",
			p2pWaiting: "Waiting for receiver, keep this page open",
			p2pSignalExchange: "Exchanging connection info",
			p2pLanCheck: "Checking LAN direct path",
			p2pInternetCheck: "Trying assisted connection",
			p2pSlow: "Direct connection is slow, still trying",
			p2pConnecting: "Connecting directly",
			p2pConnected: "Direct connection established",
			p2pSending: "Direct transfer in progress",
			p2pSent: "Direct transfer complete",
			p2pNotStored: "Direct transfer complete, not stored on server",
			p2pReceiving: "Receiving direct content",
			p2pDone: "Direct receive complete",
			p2pFailed: "Direct transfer failed",
			p2pFallback: "Switching to server transfer",
			p2pCancelled: "Direct transfer cancelled",
			p2pLinkLabel: "Direct Connection",
			p2pSenderOffline: "Sender is offline or direct connection failed",
			p2pElapsed: "Elapsed",
			p2pRouteLabel: "Path",
			p2pRouteLan: "LAN direct",
			p2pRouteNat: "NAT traversal",
			p2pRouteRelay: "TURN relay",
			p2pCandidateLocal: "Local",
			p2pCandidateRemote: "Remote",
			pullFail: "Receive failed",
			pulling: "Receiving…",
			pullOk: "Received successfully",
			enterKeyWarn: "Please enter Key",
			deleteFail: "Delete failed",
			deleteOk: "Deleted successfully",
			deleted: "Deleted",
			keyCopied: "Key copied",
			linkCopied: "Link copied",
			contentCopied: "Content copied",
			qrFail: "Failed to generate QR code",
			reqFail: "Request failed",
			metaType: "Type",
			metaSize: "Size",
			metaFile: "File",
			metaFiles: "Files",
			metaStored: "Stored",
		},
	};

	let currentLang = localStorage.getItem("cs_lang") || "zh";

	function t(key) {
		return (i18n[currentLang] && i18n[currentLang][key]) || key;
	}

	function fileCountText(count) {
		return currentLang === "zh" ? `${count} 个文件` : `${count} files`;
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
		$$("[data-i18n-aria-label]").forEach((el) => {
			const key = el.getAttribute("data-i18n-aria-label");
			if (i18n[currentLang][key] !== undefined) {
				el.setAttribute("aria-label", i18n[currentLang][key]);
			}
		});
		// Update html lang attribute
		document.documentElement.lang = currentLang === "zh" ? "zh-CN" : "en";
		updatePaperFoldLabel();
		updateSelectedFilesPreview();
		updateDirectTransferText();
		updateQuickResultToggleText();
	}

	function toggleLang() {
		currentLang = currentLang === "zh" ? "en" : "zh";
		localStorage.setItem("cs_lang", currentLang);
		applyI18n();
		updateDropZoneState();
		const btn = $("#langToggle");
		btn.textContent = currentLang === "zh" ? "EN" : "中";
		btn.title = currentLang === "zh" ? "Switch to English" : "切换到中文";
	}

	// ── Theme ──────────────────────────────────────────────

	let currentTheme = localStorage.getItem("cs_theme") || "light";

	function applyTheme(theme) {
		currentTheme = theme;
		document.documentElement.setAttribute("data-theme", theme);
		localStorage.setItem("cs_theme", theme);
		const btn = $("#themeToggle");
		btn.textContent = theme === "dark" ? "☀" : "☾";
		const metaColor = $("#metaThemeColor");
		if (metaColor) {
			metaColor.content = theme === "dark" ? "#0d1117" : "#f7f8fb";
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

	function buildShareUrl(key) {
		const url = new URL(window.location.href);
		url.search = "";
		url.hash = "";
		url.searchParams.set("key", key);
		return url.toString();
	}

	function buildP2PUrl(sessionID) {
		const url = new URL(window.location.href);
		url.search = "";
		url.hash = "";
		url.searchParams.set("p2p", sessionID);
		return url.toString();
	}

	let currentResult = { mode: "", key: "", url: "" };
	let transferMode = localStorage.getItem("cs_transfer") || "smart";
	let selectedFiles = [];
	let p2pState = null;
	let interactionMode = "modern";
	let paperOpen = false;
	let modernSendTimer = null;
	let pendingPaperFoldAfterSend = false;

	const p2pIceServers = [
		{ urls: "stun:stun.l.google.com:19302" },
		{ urls: "stun:stun.cloudflare.com:3478" },
	];
	const p2pLanFallbackDelay = 2000;
	const p2pSlowNoticeDelay = 5000;
	const p2pConnectTimeout = 10000;
	const p2pChunkSize = 64 * 1024;
	const p2pBufferLimit = 1 << 20;
	const p2pPollWaitSeconds = 25;

	// ── Init ──────────────────────────────────────────────────

	// Apply saved theme
	applyTheme(currentTheme);

	// Apply saved language
	applyI18n();
	// ── Auto-pull from URL ?key= ──────────────────────────────

	function setRadarMode(tabName) {
		const stage = $(".radar-stage");
		if (!stage) return;
		const isPull = tabName === "pull";
		stage.classList.toggle("radar-push", !isPull);
		stage.classList.toggle("radar-pull", isPull);
	}

	(function autoPullFromURL() {
		const params = new URLSearchParams(window.location.search);
		const p2pSession = params.get("p2p");
		const key = params.get("key");
		if (!key && !p2pSession) return;

		// Clean URL without reloading
		const cleanUrl = window.location.pathname + window.location.hash;
		window.history.replaceState(null, "", cleanUrl);

		// Switch to Pull tab
		$$(".tab").forEach((t) => t.classList.remove("active"));
		$$(".tab-content").forEach((c) => c.classList.remove("active"));
		$('.tab[data-tab="pull"]').classList.add("active");
		$("#pullTab").classList.add("active");
		setRadarMode("pull");

		if (p2pSession) {
			setTimeout(() => startP2PReceive(p2pSession), 0);
			return;
		}

		// Fill key and trigger pull
		$("#pullKey").value = key;
		// Defer click so all init is done
		setTimeout(() => $("#pullBtn").click(), 0);
	})();

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

	function closeSettingsPanel() {
		$("#settingsPanel").classList.add("hidden");
	}

	document.addEventListener("pointerdown", (e) => {
		const panel = $("#settingsPanel");
		if (panel.classList.contains("hidden")) return;
		if (panel.contains(e.target) || $("#settingsToggle").contains(e.target)) return;
		closeSettingsPanel();
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
			setRadarMode(tab.dataset.tab);
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
			$("#clearFile").classList.toggle("hidden", isText || !selectedFiles.length);
			syncModernPushState();
		});
	});

	function setTransferMode(mode) {
		transferMode = mode === "server" ? "server" : "smart";
		localStorage.setItem("cs_transfer", transferMode);
		$$(".transfer").forEach((btn) => {
			btn.classList.toggle("active", btn.dataset.transfer === transferMode);
		});
		const toggle = $("#directTransferToggle");
		if (toggle) {
			toggle.checked = transferMode === "smart";
		}
		updateDirectTransferText();
	}

	function updateDirectTransferText() {
		const text = $(".direct-toggle-text");
		if (!text) return;
		const key = transferMode === "smart" ? "smartTransfer" : "serverTransfer";
		text.textContent = t(key);
		const toggle = $("#directTransferToggle");
		if (toggle) {
			toggle.setAttribute("aria-label", t(key));
		}
		const label = $(".direct-toggle");
		if (label) {
			label.title = transferMode === "smart" ? t("directTransferHint") : t("serverTransfer");
		}
	}

	$$(".transfer").forEach((btn) => {
		btn.addEventListener("click", () => setTransferMode(btn.dataset.transfer));
	});

	$("#directTransferToggle").addEventListener("change", (e) => {
		setTransferMode(e.target.checked ? "smart" : "server");
	});

	setTransferMode(transferMode);

	// ── Push: file handling ───────────────────────────────────

	const dropZone = $("#dropZone");
	const fileInput = $("#fileInput");
	fileInput.multiple = true;

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
			addSelectedFiles(e.dataTransfer.files);
		}
	});

	fileInput.addEventListener("change", () => {
		if (fileInput.files.length) {
			addSelectedFiles(fileInput.files);
			fileInput.value = "";
		}
	});

	$("#clearFile").addEventListener("click", (e) => {
		e.preventDefault();
		e.stopPropagation();
		clearSelectedFile();
	});

	$("#selectedFileList").addEventListener("click", (e) => {
		const btn = e.target.closest("[data-remove-file-index]");
		if (!btn) return;
		e.preventDefault();
		e.stopPropagation();
		removeSelectedFile(Number(btn.dataset.removeFileIndex));
	});

	function updateDropZoneState() {
		const hasFile = selectedFiles.length > 0;
		dropZone.classList.toggle("has-file", hasFile);
		dropZone.classList.toggle("has-multiple-files", selectedFiles.length > 1);
		if (hasFile) {
			const totalSize = selectedFiles.reduce((sum, file) => sum + file.size, 0);
			$("#dropTitle").removeAttribute("data-i18n");
			$("#dropSubtitle").removeAttribute("data-i18n");
			$("#dropTitle").textContent = selectedFilesTitle(selectedFiles.length);
			$("#dropSubtitle").textContent = selectedFilesSubtitle(totalSize);
			return;
		}
		$("#dropTitle").setAttribute("data-i18n", "dropHint");
		$("#dropSubtitle").setAttribute("data-i18n", "dropSubHint");
		$("#dropTitle").textContent = t("dropHint");
		$("#dropSubtitle").textContent = t("dropSubHint");
	}

	function addSelectedFiles(files) {
		const seen = new Set(selectedFiles.map(fileSignature));
		for (const file of Array.from(files)) {
			const sig = fileSignature(file);
			if (seen.has(sig)) continue;
			seen.add(sig);
			selectedFiles.push(file);
		}
		updateDropZoneState();
		updateSelectedFilesPreview();
		$("#clearFile").classList.remove("hidden");
		syncModernPushState();
	}

	function fileSignature(file) {
		return `${file.name}\n${file.size}\n${file.lastModified}`;
	}

	function updateSelectedFilesPreview() {
		const list = $("#selectedFileList");
		if (!selectedFiles.length) {
			$("#packageFileName").textContent = "";
			$("#packageFileSize").textContent = "";
			list.replaceChildren();
			list.classList.add("hidden");
			return;
		}
		const totalSize = selectedFiles.reduce((sum, file) => sum + file.size, 0);
		$("#packageFileName").textContent =
			selectedFiles.length === 1
				? selectedFiles[0].name
				: fileCountText(selectedFiles.length);
		$("#packageFileSize").textContent = humanSize(totalSize);
		list.replaceChildren();
		selectedFiles.forEach((file, index) => {
			const item = document.createElement("li");
			item.className = "selected-file-item";

			const name = document.createElement("span");
			name.className = "selected-file-name";
			name.textContent = file.name;
			name.title = file.name;

			const size = document.createElement("span");
			size.className = "selected-file-size";
			size.textContent = humanSize(file.size);

			const remove = document.createElement("button");
			remove.type = "button";
			remove.className = "selected-file-remove";
			remove.dataset.removeFileIndex = String(index);
			remove.setAttribute("aria-label", `${t("removeFile")}: ${file.name}`);
			remove.title = t("removeFile");
			remove.innerHTML =
				'<svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M6 6l12 12" /><path d="M18 6L6 18" /></svg>';

			item.append(name, size, remove);
			list.appendChild(item);
		});
		list.classList.remove("hidden");
	}

	function clearSelectedFile() {
		selectedFiles = [];
		fileInput.value = "";
		updateSelectedFilesPreview();
		$("#clearFile").classList.add("hidden");
		updateDropZoneState();
		syncModernPushState();
	}

	function removeSelectedFile(index) {
		if (index < 0 || index >= selectedFiles.length) return;
		selectedFiles.splice(index, 1);
		fileInput.value = "";
		updateDropZoneState();
		updateSelectedFilesPreview();
		$("#clearFile").classList.toggle(
			"hidden",
			!selectedFiles.length || $(".mode.active")?.dataset.mode === "text",
		);
		syncModernPushState();
	}

	function selectedFilesTitle(count) {
		return currentLang === "zh"
			? `已选择 ${fileCountText(count)}`
			: `${fileCountText(count)} selected`;
	}

	function selectedFilesSubtitle(totalSize) {
		return currentLang === "zh"
			? `总计 ${humanSize(totalSize)}，可继续发送`
			: `${humanSize(totalSize)} total, ready to send`;
	}

	// ── Interaction mode ──────────────────────────────────────

	function applyInteractionMode() {
		interactionMode = "modern";
		document.documentElement.setAttribute("data-interaction", interactionMode);
		localStorage.removeItem("cs_interaction");
		syncModernPushState();
	}

	function syncModernPushState() {
		const isText = $(".mode.active")?.dataset.mode === "text";
		if (interactionMode === "modern" && isText) {
			setPaperOpen(!!$("#pushText").value.trim(), false);
		}
		updatePaperFoldLabel();
		updateModernReadyState();
	}

	function setPaperOpen(open, focusText) {
		const textMode = $("#textMode");
		const wasOpen = paperOpen;
		paperOpen = !!open;
		textMode.classList.remove("paper-folding");
		textMode.classList.toggle("paper-open", paperOpen);
		if (wasOpen && !paperOpen) {
			void textMode.offsetWidth;
			textMode.classList.add("paper-folding");
			setTimeout(() => textMode.classList.remove("paper-folding"), 420);
		}
		updatePaperFoldLabel();
		if (paperOpen && focusText) {
			requestAnimationFrame(() => $("#pushText").focus());
		}
	}

	function updatePaperFoldLabel() {
		const text = $("#pushText")?.value.trim() || "";
		const preview = $(".paper-preview");
		const countBadge = $(".paper-count");
		$("#paperPrompt").classList.toggle("has-paper-text", !!text);
		if (!text) {
			if (preview) preview.textContent = "";
			if (countBadge) countBadge.textContent = "";
			return;
		}

		const count = Array.from(text).length;
		if (countBadge) countBadge.textContent = String(count);
		if (preview) {
			const summary = text.replace(/\s+/g, " ");
			preview.textContent =
				Array.from(summary).slice(0, 42).join("") +
				(count > 42 ? "..." : "");
		}
	}

	function updateModernReadyState() {
		const btn = $("#pushBtn");
		if (!btn || btn.dataset.busy === "true") return;
		if (interactionMode !== "modern") {
			btn.disabled = false;
			btn.classList.remove("modern-ready");
			return;
		}

		const isText = $(".mode.active")?.dataset.mode === "text";
		const ready = isText ? !!$("#pushText").value.trim() : selectedFiles.length > 0;
		btn.disabled = !ready;
		btn.classList.toggle("modern-ready", ready);
	}

	function triggerModernSendAnimation(type) {
		if (interactionMode !== "modern") return;
		const target = type === "text" ? $("#textMode") : $("#fileMode");
		if (modernSendTimer) {
			clearTimeout(modernSendTimer);
			modernSendTimer = null;
		}
		if (type === "text") pendingPaperFoldAfterSend = false;
		target.classList.remove("modern-sending");
		void target.offsetWidth;
		target.classList.add("modern-sending");
		modernSendTimer = setTimeout(() => {
			if (type === "text" && pendingPaperFoldAfterSend && paperOpen) {
				setPaperOpen(false, false);
			}
			if (type === "text") pendingPaperFoldAfterSend = false;
			target.classList.remove("modern-sending");
			modernSendTimer = null;
		}, 900);
	}

	function foldPaperAfterPush(input) {
		if (interactionMode !== "modern" || input?.type !== "text" || !paperOpen) return;
		const textMode = $("#textMode");
		if (textMode.classList.contains("modern-sending")) {
			// 发送动画还在收尾时，先记下收起动作，避免先展开再折叠。
			pendingPaperFoldAfterSend = true;
			return;
		}
		setPaperOpen(false, false);
	}

	$("#paperPrompt").addEventListener("click", () => setPaperOpen(true, true));
	$("#paperCollapse").addEventListener("click", () => setPaperOpen(false, false));

	$("#pushText").addEventListener("input", () => {
		if (interactionMode === "modern" && $("#pushText").value.trim()) {
			setPaperOpen(true, false);
		}
		updatePaperFoldLabel();
		updateModernReadyState();
	});

	document.addEventListener("pointerdown", (e) => {
		if (interactionMode !== "modern" || !paperOpen) return;
		if (!$("#pushTab").classList.contains("active")) return;
		if ($(".mode.active")?.dataset.mode !== "text") return;
		if ($("#textMode").contains(e.target)) return;
		if ($("#pushText").value.trim()) return;

		setPaperOpen(false, false);
	});

	applyInteractionMode();

	// ── Push ──────────────────────────────────────────────────

	$("#pushBtn").addEventListener("click", async () => {
		const btn = $("#pushBtn");
		const isText = $(".mode.active").dataset.mode === "text";

		try {
			const input = getPushInput(isText);
			if (!input) return;
			const pushMore = $(".push-more");
			if (pushMore) pushMore.open = false;
			btn.dataset.busy = "true";
			btn.disabled = true;
			btn.textContent = t("pushing");
			triggerModernSendAnimation(input.type);
			clearPushResult();

			if (transferMode === "smart" && input.type === "files") {
				await pushToServer(input);
				return;
			}

			if (transferMode === "smart") {
				if (!supportsP2P()) {
					toast(t("p2pUnsupported"), "error");
					await pushToServer(input);
					return;
				}
				await startP2PPush(input);
				return;
			}

			await pushToServer(input);
		} catch (e) {
			toast(t("reqFail") + ": " + e.message, "error");
		} finally {
			delete btn.dataset.busy;
			btn.disabled = false;
			btn.textContent = t("push");
			updateModernReadyState();
		}
	});

	function getPushInput(isText) {
		const ttl = parseInt($("#pushTtl").value);
		if (isText) {
			const text = $("#pushText").value;
			if (!text.trim()) {
				toast(t("enterText"), "error");
				return null;
			}
			return {
				type: "text",
				text,
				ttl,
				size: new Blob([text]).size,
				contentType: "text/plain; charset=utf-8",
			};
		}

		if (!selectedFiles.length) {
			toast(t("selectFile"), "error");
			return null;
		}
		const totalSize = selectedFiles.reduce((sum, file) => sum + file.size, 0);
		const singleFile = selectedFiles.length === 1 ? selectedFiles[0] : null;
		return {
			type: singleFile ? "binary" : "files",
			file: singleFile,
			files: [...selectedFiles],
			ttl,
			size: totalSize,
			filename: singleFile ? singleFile.name : "crossshare-files.zip",
			contentType: singleFile
				? singleFile.type || "application/octet-stream"
				: "application/zip",
		};
	}

	function clearPushResult() {
		stopCountdown();
		currentResult = { mode: "", key: "", url: "" };
		$("#resultKey").textContent = "";
		$("#resultMeta").textContent = "";
		$("#pushResult").classList.add("hidden");
		$("#pushQuickResult").classList.add("hidden");
		$("#quickResultValue").textContent = "";
		updateQuickResultToggleText();
		setPushResultActions("");
	}

	async function pushToServer(input) {
		cancelP2P(false);

		try {
			const result = await uploadToServer(input);
			renderServerPushResult(result);
			toast(t("pushOk"), "success");
			foldPaperAfterPush(input);
		} catch (e) {
			toast(`${t("pushFail")}: ${e.message}`, "error");
		}
	}

	async function uploadToServer(input) {
		let resp;
		if (input.type === "text") {
			const body = { text: input.text };
			if (input.ttl > 0) body.ttl = input.ttl;

			resp = await fetch(apiUrl("/push/text"), {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...authHeaders(),
				},
				body: JSON.stringify(body),
			});
		} else {
			const body = new FormData();
			for (const file of input.files || [input.file]) {
				body.append("files", file, file.name);
			}
			if (input.ttl > 0) body.append("ttl", String(input.ttl));
			if (input.type === "files" && input.filename) {
				body.append("name", input.filename);
			}

			resp = await fetch(apiUrl("/push/files"), {
				method: "POST",
				headers: authHeaders(),
				body,
			});
		}

		const data = await resp.json();
		if (data.code !== 0) {
			throw new Error(data.msg || t("pushFail"));
		}

		return data.data;
	}

	function renderServerPushResult(r) {
		currentResult = {
			mode: "server",
			key: r.key,
			url: buildShareUrl(r.key),
		};
		$("#resultKey").textContent = r.key;
		let meta = `${t("metaType")}: ${r.type} · ${t("metaSize")}: ${humanSize(r.size)} · TTL: ${humanDuration(r.ttl)}`;
		if (r.filename) meta += ` · ${t("metaFile")}: ${r.filename}`;
		if (r.file_count) meta += ` · ${t("metaFiles")}: ${r.file_count}`;
		if (r.stored_size && r.stored_size !== r.size) {
			meta += ` · ${t("metaStored")}: ${humanSize(r.stored_size)}`;
		}
		$("#resultMeta").textContent = meta;
		$("#pushResult").classList.add("hidden");
		renderQuickPushResult("server", t("pushOk"), `Key: ${r.key}`);
		setPushResultActions("server");

		// Start countdown timer
		startCountdown(r.expire_at || Math.floor(Date.now() / 1000) + r.ttl);
	}

	function renderQuickPushResult(mode, status, value) {
		$("#quickResultStatus").textContent = status;
		$("#quickResultValue").textContent = value || "";
		$("#pushQuickResult").classList.remove("hidden");
		setPushResultActions(mode);
		updateQuickResultToggleText();
		requestAnimationFrame(() => {
			$("#pushQuickResult").scrollIntoView({
				behavior: "smooth",
				block: "nearest",
			});
		});
	}

	function supportsP2P() {
		return (
			window.RTCPeerConnection &&
			window.RTCSessionDescription &&
			window.RTCIceCandidate
		);
	}

	function setPushResultActions(mode) {
		const hasShareLink = mode === "server" || mode === "p2p";
		$("#copyKey").classList.toggle("hidden", mode !== "server");
		$("#copyLink").classList.toggle("hidden", !hasShareLink);
		$("#showQr").classList.toggle("hidden", !hasShareLink);
		$("#cleanupServerPush").classList.toggle("hidden", mode !== "server");
		$("#saveToServer").classList.toggle("hidden", mode !== "p2p");
		$("#cancelP2p").classList.toggle("hidden", mode !== "p2p");
		$("#quickCopyKey").classList.toggle("hidden", mode !== "server");
		$("#quickCopyLink").classList.toggle("hidden", !hasShareLink);
		$("#quickShowQr").classList.toggle("hidden", !hasShareLink);
		$("#quickToggleResult").classList.toggle("hidden", mode !== "server");
		updateQuickResultToggleText();
	}

	function updateP2PStatus(target, message, active, detail) {
		const state = p2pState;
		if (state) {
			state.statusTarget = target;
			state.statusMessage = message;
			state.statusActive = active;
			state.statusDetail = detail;
		}
		renderP2PStatus(target, message, active, detail, state?.startedAt, state?.p2pDiagnostic);
		if (!state) return;

		if (active && !state.statusTimer) {
			state.statusTimer = setInterval(() => {
				if (p2pState !== state || !state.statusActive) {
					stopP2PStatusTimer(state);
					return;
				}
				renderP2PStatus(
					state.statusTarget,
					state.statusMessage,
					state.statusActive,
					state.statusDetail,
					state.startedAt,
					state.p2pDiagnostic,
				);
			}, 500);
		}
		if (!active) {
			stopP2PStatusTimer(state);
		}
	}

	function renderP2PStatus(target, message, active, detail, startedAt, diagnostic) {
		const el = target === "pull" ? $("#pullMeta") : $("#resultMeta");
		if (!el) return;

		const elapsed = startedAt ? formatElapsed(Date.now() - startedAt) : "";
		const fullDetail = [detail, diagnostic].filter(Boolean).join(" · ");
		const indicator = active
			? '<span class="p2p-spinner"></span>'
			: '<span class="p2p-dot"></span>';
		el.innerHTML = `
			<div class="p2p-status ${active ? "is-active" : ""}">
				${indicator}
				<span class="p2p-status-main">${escapeHTML(message)}</span>
				${elapsed ? `<span class="p2p-elapsed">${escapeHTML(t("p2pElapsed"))}: ${elapsed}</span>` : ""}
			</div>
			${fullDetail ? `<div class="p2p-status-detail">${escapeHTML(fullDetail)}</div>` : ""}
		`;
	}

	function formatElapsed(ms) {
		return (ms / 1000).toFixed(1) + "s";
	}

	function escapeHTML(value) {
		return String(value).replace(/[&<>"']/g, (ch) => ({
			"&": "&amp;",
			"<": "&lt;",
			">": "&gt;",
			'"': "&quot;",
			"'": "&#39;",
		})[ch]);
	}

	async function updateP2PDiagnostic(state, target) {
		if (!state || !state.pc || p2pState !== state) return;
		const diagnostic = await getP2PDiagnostic(state.pc);
		if (!diagnostic || p2pState !== state) return;
		state.p2pDiagnostic = diagnostic;
		sendP2PDiagnostic(state);
		renderP2PStatus(
			state.statusTarget || target,
			state.statusMessage || t("p2pConnected"),
			state.statusActive !== false,
			state.statusDetail || "",
			state.startedAt,
			state.p2pDiagnostic,
		);
	}

	function scheduleP2PDiagnostic(state, target) {
		updateP2PDiagnostic(state, target).catch(() => {});
		setTimeout(() => updateP2PDiagnostic(state, target).catch(() => {}), 500);
		setTimeout(() => updateP2PDiagnostic(state, target).catch(() => {}), 1500);
	}

	function sendP2PDiagnostic(state) {
		if (
			!state ||
			state.role !== "sender" ||
			state.p2pDiagnosticSent ||
			!state.dc ||
			state.dc.readyState !== "open" ||
			!state.p2pDiagnostic
		) {
			return;
		}
		state.p2pDiagnosticSent = true;
		try {
			state.dc.send(JSON.stringify({
				kind: "diagnostic",
				diagnostic: state.p2pDiagnostic,
			}));
		} catch {
			// ignore
		}
	}

	// 读取已选 ICE candidate pair，用于展示实际 P2P 路径。
	async function getP2PDiagnostic(pc) {
		if (!pc.getStats) return "";
		const stats = await pc.getStats();
		let selectedPair = null;

		stats.forEach((report) => {
			if (report.type === "transport" && report.selectedCandidatePairId) {
				selectedPair = stats.get(report.selectedCandidatePairId);
			}
		});
		if (!selectedPair) {
			let succeededPair = null;
			let succeededPairBytes = -1;
			stats.forEach((report) => {
				if (
					report.type === "candidate-pair" &&
					(report.selected || (report.nominated && report.state === "succeeded"))
				) {
					selectedPair = report;
				}
				if (report.type === "candidate-pair" && report.state === "succeeded") {
					const bytes = (report.bytesSent || 0) + (report.bytesReceived || 0);
					if (bytes > succeededPairBytes) {
						succeededPair = report;
						succeededPairBytes = bytes;
					}
				}
			});
			if (!selectedPair) selectedPair = succeededPair;
		}
		if (!selectedPair) return "";

		const local = stats.get(selectedPair.localCandidateId);
		const remote = stats.get(selectedPair.remoteCandidateId);
		if (!local || !remote) return "";

		const localType = local.candidateType || "";
		const remoteType = remote.candidateType || "";
		const route = routeTypeText(localType, remoteType);
		const protocol = local.protocol || remote.protocol || "";
		const parts = [
			`${t("p2pRouteLabel")}: ${route}`,
			`${t("p2pCandidateLocal")}: ${candidateTypeText(localType)}`,
			`${t("p2pCandidateRemote")}: ${candidateTypeText(remoteType)}`,
		];
		if (protocol) parts.push(protocol.toUpperCase());
		return parts.join(" · ");
	}

	function routeTypeText(localType, remoteType) {
		if (localType === "relay" || remoteType === "relay") return t("p2pRouteRelay");
		if (localType === "host" && remoteType === "host") return t("p2pRouteLan");
		return t("p2pRouteNat");
	}

	function candidateTypeText(type) {
		const labels =
			currentLang === "zh"
				? { host: "主机", srflx: "公网映射", prflx: "对端反射", relay: "中继" }
				: { host: "host", srflx: "srflx", prflx: "prflx", relay: "relay" };
		return labels[type] || type || "-";
	}

	async function startP2PPush(input) {
		cancelP2P(false);

		const resp = await fetch(apiUrl("/p2p/sessions"), {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				...authHeaders(),
			},
			body: "{}",
		});
		const data = await resp.json();
		if (data.code !== 0) {
			toast(`${t("pushFail")}: ${data.msg}`, "error");
			return;
		}

		const sessionID = data.data.session_id;
		const shareUrl = buildP2PUrl(sessionID);
		currentResult = { mode: "p2p", key: sessionID, url: shareUrl };
		$("#resultKey").textContent = t("p2pLinkLabel");
		$("#pushResult").classList.remove("hidden");
		renderQuickPushResult("p2p", t("p2pLinkLabel"), t("p2pWaiting"));
		setPushResultActions("p2p");

		p2pState = {
			role: "sender",
			sessionID,
			pc: null,
			dc: null,
			input,
			lastSeq: 0,
			stopped: false,
			connected: false,
			transferStarted: false,
			transferDone: false,
			fallbackStarted: false,
			attempt: 0,
			attemptMode: "",
			connectTimer: null,
			upgradeTimer: null,
			slowTimer: null,
			statusTimer: null,
			pollController: null,
			p2pDiagnostic: "",
			p2pDiagnosticSent: false,
			startedAt: Date.now(),
			pendingCandidates: [],
			futureCandidates: {},
		};

		pollP2PMessages("sender");
		updateP2PStatus("push", t("p2pWaiting"), true, shareUrl);
		await startP2PSenderAttempt("lan");
		toast(t("p2pWaiting"), "success");
		foldPaperAfterPush(input);
	}

	async function startP2PSenderAttempt(mode) {
		const state = p2pState;
		if (!state || state.role !== "sender" || state.stopped || state.connected || state.transferDone) return;
		if (state.attemptMode === mode && state.pc) return;

		const attempt = mode === "lan" ? 1 : 2;
		state.attempt = attempt;
		state.attemptMode = mode;
		state.pendingCandidates = state.futureCandidates[attempt] || [];
		delete state.futureCandidates[attempt];
		clearP2PTimers(state);
		closeP2PConnection(state);
		updateP2PStatus("push", mode === "lan" ? t("p2pLanCheck") : t("p2pInternetCheck"), true, currentResult.url);

		const pc = new RTCPeerConnection({ iceServers: mode === "lan" ? [] : p2pIceServers });
		const dc = pc.createDataChannel("crossshare");
		state.pc = pc;
		state.dc = dc;

		pc.onicecandidate = (event) => {
			if (event.candidate && p2pState === state && state.attempt === attempt) {
				postP2PMessage(state.sessionID, "sender", "receiver", "candidate", packP2PSignal(attempt, mode, event.candidate.toJSON())).catch(() => {});
			}
		};
		pc.onconnectionstatechange = () => {
			if (p2pState !== state || state.attempt !== attempt || state.transferDone) return;
			if (pc.connectionState === "connected") {
				state.connected = true;
				clearP2PTimers(state);
				updateP2PStatus("push", t("p2pConnected"), true, currentResult.url);
				scheduleP2PDiagnostic(state, "push");
			}
			if (pc.connectionState === "failed" || pc.connectionState === "disconnected") {
				if (!state.connected && mode === "lan") {
					startP2PSenderAttempt("stun").catch(() => markP2PFailed());
					return;
				}
				markP2PFailed();
			}
		};
		dc.onopen = () => {
			if (p2pState !== state || state.attempt !== attempt) return;
			state.connected = true;
			clearP2PTimers(state);
			updateP2PStatus("push", t("p2pConnected"), true, currentResult.url);
			scheduleP2PDiagnostic(state, "push");
			sendP2PContent(state.input).catch((e) => {
				toast(t("reqFail") + ": " + e.message, "error");
				markP2PFailed();
			});
		};
		dc.onerror = () => {
			if (p2pState === state && state.attempt === attempt) markP2PFailed();
		};

		const offer = await pc.createOffer();
		await pc.setLocalDescription(offer);
		await postP2PMessage(state.sessionID, "sender", "receiver", "offer", packP2PSignal(attempt, mode, describeSession(pc.localDescription)));
	}

	function armP2PConnectTimers(attempt) {
		const state = p2pState;
		if (!state || state.role !== "sender" || state.connected || state.transferDone) return;
		clearP2PTimers(state);

		if (attempt === 1) {
			state.upgradeTimer = setTimeout(() => {
				if (p2pState === state && !state.connected) {
					startP2PSenderAttempt("stun").catch(() => markP2PFailed());
				}
			}, p2pLanFallbackDelay);
		}
		state.slowTimer = setTimeout(() => {
			if (p2pState === state && !state.connected) {
				updateP2PStatus("push", t("p2pSlow"), true, currentResult.url);
			}
		}, p2pSlowNoticeDelay);
		state.connectTimer = setTimeout(() => {
			if (p2pState === state && !state.connected) {
				markP2PFailed();
			}
		}, p2pConnectTimeout);
	}

	async function sendP2PContent(input) {
		if (!p2pState || !p2pState.dc || p2pState.stopped) return;

		const state = p2pState;
		const dc = p2pState.dc;
		state.connected = true;
		state.transferStarted = true;
		clearP2PTimers(state);
		updateP2PStatus("push", t("p2pSending"), true, currentResult.url);
		dc.send(JSON.stringify({
			kind: "meta",
			type: input.type,
			filename: input.filename || "",
			content_type: input.contentType,
			size: input.size,
		}));

		const blob = input.type === "text"
			? new Blob([input.text], { type: input.contentType })
			: input.file;
		for (let offset = 0; offset < blob.size; offset += p2pChunkSize) {
			if (!p2pState || p2pState.stopped) return;
			await waitDataChannelBuffer(dc);
			dc.send(await blob.slice(offset, offset + p2pChunkSize).arrayBuffer());
		}

		await updateP2PDiagnostic(state, "push");
		dc.send(JSON.stringify({ kind: "done", diagnostic: state.p2pDiagnostic || "" }));
		state.transferDone = true;
		state.stopped = true;
		setPushResultActions("p2pDone");
		renderQuickPushResult("p2pDone", t("p2pSent"), t("p2pNotStored"));
		updateP2PStatus("push", t("p2pNotStored"), false, currentResult.url);
		toast(t("p2pSent"), "success");
		foldPaperAfterPush(input);
	}

	function waitDataChannelBuffer(dc) {
		if (dc.bufferedAmount < p2pBufferLimit) {
			return Promise.resolve();
		}
		dc.bufferedAmountLowThreshold = p2pBufferLimit / 2;
		return new Promise((resolve) => {
			dc.onbufferedamountlow = () => {
				dc.onbufferedamountlow = null;
				resolve();
			};
		});
	}

	function markP2PFailed() {
		if (!p2pState || p2pState.transferDone) return;
		updateP2PStatus("push", t("p2pFailed"), false, currentResult.url);
		if (p2pState.transferStarted || p2pState.connected) {
			toast(t("p2pFailed"), "error");
			return;
		}
		fallbackP2PToServer();
	}

	async function fallbackP2PToServer() {
		const state = p2pState;
		if (!state || state.role !== "sender" || !state.input || state.connected || state.transferStarted || state.transferDone || state.fallbackStarted) return;
		state.fallbackStarted = true;
		stopP2PTransport(state);
		state.stopped = true;
		updateP2PStatus("push", t("p2pFallback"), true, currentResult.url);

		try {
			const result = await uploadToServer(state.input);
			await postP2PMessage(state.sessionID, "sender", "receiver", "fallback", {
				key: result.key,
			}).catch(() => {});
			renderServerPushResult(result);
			p2pState = null;
			toast(t("pushOk"), "success");
			foldPaperAfterPush(state.input);
		} catch (e) {
			state.fallbackStarted = false;
			updateP2PStatus("push", t("p2pFailed"), false, currentResult.url);
			toast(`${t("pushFail")}: ${e.message}`, "error");
		}
	}

	function closeP2PConnection(state) {
		if (state.dc) {
			state.dc.onopen = null;
			state.dc.onerror = null;
			state.dc.close();
			state.dc = null;
		}
		if (state.pc) {
			state.pc.onicecandidate = null;
			state.pc.onconnectionstatechange = null;
			state.pc.ondatachannel = null;
			state.pc.close();
			state.pc = null;
		}
	}

	function clearP2PTimers(state) {
		if (state.connectTimer) clearTimeout(state.connectTimer);
		if (state.upgradeTimer) clearTimeout(state.upgradeTimer);
		if (state.slowTimer) clearTimeout(state.slowTimer);
		state.connectTimer = null;
		state.upgradeTimer = null;
		state.slowTimer = null;
	}

	function stopP2PStatusTimer(state) {
		if (state.statusTimer) clearInterval(state.statusTimer);
		state.statusTimer = null;
	}

	function stopP2PPoll(state) {
		if (state.pollController) {
			state.pollController.abort();
			state.pollController = null;
		}
	}

	function stopP2PTransport(state) {
		clearP2PTimers(state);
		stopP2PStatusTimer(state);
		stopP2PPoll(state);
		closeP2PConnection(state);
	}

	function cancelP2P(showToast) {
		if (!p2pState) return;
		const state = p2pState;
		state.stopped = true;
		stopP2PTransport(state);
		if (state.sessionID) {
			fetch(apiUrl("/p2p/sessions/" + state.sessionID), {
				method: "DELETE",
				headers: authHeaders(),
			}).catch(() => {});
		}
		p2pState = null;
		if (showToast) {
			$("#resultMeta").textContent = t("p2pCancelled");
			toast(t("p2pCancelled"), "success");
		}
	}

	async function postP2PMessage(sessionID, from, to, type, payload) {
		const resp = await fetch(apiUrl("/p2p/sessions/" + sessionID + "/messages"), {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				...authHeaders(),
			},
			body: JSON.stringify({ from, to, type, payload }),
		});
		const data = await resp.json();
		if (data.code !== 0) {
			throw new Error(data.msg || "p2p signal failed");
		}
		return data.data;
	}

	async function pollP2PMessages(role) {
		while (p2pState && !p2pState.stopped && p2pState.role === role) {
			const state = p2pState;
			const sessionID = state.sessionID;
			const after = state.lastSeq;
			const controller = new AbortController();
			state.pollController = controller;
			try {
				const resp = await fetch(
					apiUrl(`/p2p/sessions/${sessionID}/messages?to=${role}&after=${after}&wait=${p2pPollWaitSeconds}`),
					{ headers: authHeaders(), signal: controller.signal },
				);
				if (state.pollController === controller) {
					state.pollController = null;
				}
				if (resp.status === 204) return;
				const data = await resp.json();
				if (data.code !== 0) {
					if (p2pState && p2pState.role === "receiver") {
						showP2PReceiveError(t("p2pSenderOffline"));
					}
					return;
				}

				for (const msg of data.data.messages || []) {
					if (!p2pState || p2pState.stopped) return;
					p2pState.lastSeq = Math.max(p2pState.lastSeq, msg.seq);
					await handleP2PMessage(msg);
				}
			} catch (e) {
				if (state.pollController === controller) {
					state.pollController = null;
				}
				if (e.name === "AbortError") return;
				if (p2pState && !p2pState.stopped) {
					if (role === "sender") markP2PFailed();
					if (role === "receiver") showP2PReceiveError(t("p2pSenderOffline"));
				}
				return;
			}
		}
	}

	async function handleP2PMessage(msg) {
		if (!p2pState) return;

		if (msg.type === "fallback" && p2pState.role === "receiver" && msg.payload?.key) {
			const state = p2pState;
			stopP2PTransport(state);
			state.stopped = true;
			p2pState = null;
			$("#pullKey").value = msg.payload.key;
			$("#pullMeta").textContent = t("p2pFallback");
			$("#pullBtn").click();
			return;
		}

		if (msg.type === "offer" && p2pState.role === "receiver") {
			const signal = unpackP2PSignal(msg.payload);
			if (signal.attempt < p2pState.attempt) return;
			await startP2PReceiverAttempt(signal.attempt, signal.mode);
			const pc = p2pState.pc;
			updateP2PStatus("pull", t("p2pSignalExchange"), true, "");
			await pc.setRemoteDescription(new RTCSessionDescription(signal.data));
			await flushP2PCandidates();
			const answer = await pc.createAnswer();
			await pc.setLocalDescription(answer);
			await postP2PMessage(p2pState.sessionID, "receiver", "sender", "answer", packP2PSignal(p2pState.attempt, p2pState.attemptMode, describeSession(pc.localDescription)));
			return;
		}

		if (msg.type === "answer" && p2pState.role === "sender") {
			const signal = unpackP2PSignal(msg.payload);
			if (signal.attempt !== p2pState.attempt || !p2pState.pc) return;
			updateP2PStatus("push", t("p2pSignalExchange"), true, currentResult.url);
			await p2pState.pc.setRemoteDescription(new RTCSessionDescription(signal.data));
			await flushP2PCandidates();
			armP2PConnectTimers(signal.attempt);
			return;
		}

		if (msg.type === "candidate" && msg.payload) {
			const signal = unpackP2PSignal(msg.payload);
			if (signal.attempt < p2pState.attempt) return;
			if (signal.attempt > p2pState.attempt) {
				if (!p2pState.futureCandidates) p2pState.futureCandidates = {};
				if (!p2pState.futureCandidates[signal.attempt]) p2pState.futureCandidates[signal.attempt] = [];
				p2pState.futureCandidates[signal.attempt].push(signal.data);
				return;
			}
			if (!p2pState.pc || !p2pState.pc.remoteDescription) {
				p2pState.pendingCandidates.push(signal.data);
				return;
			}
			await p2pState.pc.addIceCandidate(new RTCIceCandidate(signal.data));
		}
	}

	async function flushP2PCandidates() {
		if (!p2pState || !p2pState.pc || !p2pState.pendingCandidates.length) return;
		const candidates = p2pState.pendingCandidates.splice(0);
		for (const candidate of candidates) {
			await p2pState.pc.addIceCandidate(new RTCIceCandidate(candidate));
		}
	}

	function packP2PSignal(attempt, mode, data) {
		return { attempt, mode, data };
	}

	function unpackP2PSignal(payload) {
		if (payload && typeof payload.attempt === "number" && payload.data) {
			return {
				attempt: payload.attempt,
				mode: payload.mode || "stun",
				data: payload.data,
			};
		}
		return { attempt: 1, mode: "stun", data: payload };
	}

	function describeSession(description) {
		return { type: description.type, sdp: description.sdp };
	}

	function copyCurrentKey() {
		if (!currentResult.key) return;
		copyText(currentResult.key);
		toast(t("keyCopied"), "success");
	}

	function copyCurrentLink() {
		const key = currentResult.key || $("#resultKey").textContent;
		copyText(currentResult.url || buildShareUrl(key));
		toast(t("linkCopied"), "success");
	}

	function showCurrentQr() {
		const key = currentResult.key || $("#resultKey").textContent;
		const shareUrl = currentResult.url || buildShareUrl(key);
		const canvas = $("#qrCanvas");

		try {
			window.CrossShareQR.render(canvas, shareUrl, {
				size: 300,
				errorCorrectionLevel: "M",
			});
		} catch (e) {
			toast(t("qrFail") + ": " + e.message, "error");
			return;
		}

		$("#qrLink").textContent = shareUrl;
		$("#qrModal").classList.remove("hidden");
	}

	function togglePushResultDetails() {
		if (!currentResult.mode) return;
		const result = $("#pushResult");
		const shouldShow = result.classList.contains("hidden");
		result.classList.toggle("hidden", !shouldShow);
		updateQuickResultToggleText();
		if (shouldShow) {
			requestAnimationFrame(() => {
				result.scrollIntoView({
					behavior: "smooth",
					block: "nearest",
				});
			});
		}
	}

	function updateQuickResultToggleText() {
		const btn = $("#quickToggleResult");
		if (!btn) return;
		btn.textContent = $("#pushResult").classList.contains("hidden")
			? t("details")
			: t("hideDetails");
	}

	$("#copyKey").addEventListener("click", copyCurrentKey);
	$("#copyLink").addEventListener("click", copyCurrentLink);
	$("#showQr").addEventListener("click", showCurrentQr);
	$("#quickCopyKey").addEventListener("click", copyCurrentKey);
	$("#quickCopyLink").addEventListener("click", copyCurrentLink);
	$("#quickShowQr").addEventListener("click", showCurrentQr);
	$("#quickToggleResult").addEventListener("click", togglePushResultDetails);

	$("#cleanupServerPush").addEventListener("click", async () => {
		if (currentResult.mode !== "server" || !currentResult.key) return;

		const btn = $("#cleanupServerPush");
		btn.disabled = true;

		try {
			const resp = await fetch(apiUrl("/pull/" + currentResult.key), {
				method: "DELETE",
				headers: authHeaders(),
			});
			const data = await resp.json();
			if (data.code !== 0) {
				toast(`${t("deleteFail")}: ${data.msg}`, "error");
				return;
			}

			clearPushResult();
			toast(t("deleteOk"), "success");
		} catch (e) {
			toast(t("reqFail") + ": " + e.message, "error");
		} finally {
			btn.disabled = false;
		}
	});

	function closeQrModal() {
		$("#qrModal").classList.add("hidden");
	}

	$("#closeQr").addEventListener("click", closeQrModal);
	$("#qrModalBackdrop").addEventListener("click", closeQrModal);
	$("#saveToServer").addEventListener("click", async () => {
		if (!p2pState || !p2pState.input) return;
		await fallbackP2PToServer();
	});
	$("#cancelP2p").addEventListener("click", () => cancelP2P(true));

	async function startP2PReceive(sessionID) {
		if (!supportsP2P()) {
			showP2PReceiveError(t("p2pUnsupported"));
			return;
		}

		cancelP2P(false);
		$("#pullTextResult").classList.add("hidden");
		$("#pullFileResult").classList.add("hidden");
		$("#pullResult").classList.remove("hidden");

		p2pState = {
			role: "receiver",
			sessionID,
			pc: null,
			dc: null,
			lastSeq: 0,
			stopped: false,
			connected: false,
			attempt: 0,
			attemptMode: "",
			statusTimer: null,
			pollController: null,
			p2pDiagnostic: "",
			startedAt: Date.now(),
			receiveMeta: null,
			receiveChunks: [],
			receivedSize: 0,
			pendingCandidates: [],
			futureCandidates: {},
		};

		updateP2PStatus("pull", t("p2pSignalExchange"), true, "");
		pollP2PMessages("receiver");
	}

	async function startP2PReceiverAttempt(attempt, mode) {
		const state = p2pState;
		if (!state || state.role !== "receiver" || state.stopped) return;
		if (attempt < state.attempt) return;
		updateP2PStatus("pull", mode === "lan" ? t("p2pLanCheck") : t("p2pInternetCheck"), true, "");
		if (attempt === state.attempt && state.attemptMode === mode && state.pc) return;

		if (attempt !== state.attempt || state.attemptMode !== mode || !state.pc) {
			state.attempt = attempt;
			state.attemptMode = mode;
			state.pendingCandidates = state.futureCandidates[attempt] || [];
			delete state.futureCandidates[attempt];
			closeP2PConnection(state);
		}

		const pc = new RTCPeerConnection({ iceServers: mode === "lan" ? [] : p2pIceServers });
		state.pc = pc;

		pc.onicecandidate = (event) => {
			if (event.candidate && p2pState === state && state.attempt === attempt) {
				postP2PMessage(state.sessionID, "receiver", "sender", "candidate", packP2PSignal(attempt, mode, event.candidate.toJSON())).catch(() => {});
			}
		};
		pc.onconnectionstatechange = () => {
			if (p2pState !== state || state.attempt !== attempt) return;
			if (pc.connectionState === "connected") {
				state.connected = true;
				updateP2PStatus("pull", t("p2pConnected"), true, "");
				scheduleP2PDiagnostic(state, "pull");
			}
			if (pc.connectionState === "failed" || pc.connectionState === "disconnected") {
				updateP2PStatus("pull", t("p2pSlow"), true, "");
			}
		};
		pc.ondatachannel = (event) => {
			if (p2pState !== state || state.attempt !== attempt) return;
			p2pState.dc = event.channel;
			p2pState.dc.binaryType = "arraybuffer";
			p2pState.dc.onmessage = handleP2PDataMessage;
			p2pState.dc.onerror = () => showP2PReceiveError(t("p2pSenderOffline"));
		};
	}

	function handleP2PDataMessage(event) {
		if (!p2pState || p2pState.role !== "receiver") return;

		if (typeof event.data === "string") {
			const msg = JSON.parse(event.data);
			if (msg.kind === "diagnostic" && msg.diagnostic) {
				p2pState.p2pDiagnostic = msg.diagnostic;
				renderP2PStatus(
					p2pState.statusTarget || "pull",
					p2pState.statusMessage || t("p2pConnected"),
					p2pState.statusActive !== false,
					p2pState.statusDetail || "",
					p2pState.startedAt,
					p2pState.p2pDiagnostic,
				);
				return;
			}
			if (msg.kind === "meta") {
				p2pState.receiveMeta = msg;
				p2pState.receiveChunks = [];
				p2pState.receivedSize = 0;
				updateP2PStatus("pull", `${t("p2pReceiving")} · ${t("metaSize")}: ${humanSize(msg.size || 0)}`, true, "");
				return;
			}
			if (msg.kind === "done") {
				if (msg.diagnostic) {
					p2pState.p2pDiagnostic = msg.diagnostic;
				}
				finishP2PReceive();
			}
			return;
		}

		p2pState.receiveChunks.push(event.data);
		p2pState.receivedSize += event.data.byteLength;
		if (p2pState.receiveMeta) {
			updateP2PStatus("pull", `${t("p2pReceiving")} · ${humanSize(p2pState.receivedSize)} / ${humanSize(p2pState.receiveMeta.size || 0)}`, true, "");
		}
	}

	function finishP2PReceive() {
		if (!p2pState || !p2pState.receiveMeta) return;
		const meta = p2pState.receiveMeta;
		const blob = new Blob(p2pState.receiveChunks, {
			type: meta.content_type || "application/octet-stream",
		});
		p2pState.stopped = true;
		stopP2PStatusTimer(p2pState);

		if (meta.type === "text") {
			blob.text().then((text) => {
				$("#pullTextContent").textContent = text;
				$("#pullTextResult").classList.remove("hidden");
				$("#pullFileResult").classList.add("hidden");
				updateP2PStatus("pull", `${t("p2pDone")} · ${t("metaSize")}: ${humanSize(blob.size)}`, false, "");
				scheduleP2PDiagnostic(p2pState, "pull");
				toast(t("p2pDone"), "success");
			});
			return;
		}

		const url = URL.createObjectURL(blob);
		$("#pullFileName").textContent = meta.filename || "download";
		const link = $("#pullFileLink");
		link.href = url;
		link.download = meta.filename || "download";
		link.textContent = t("download");
		$("#pullFileResult").classList.remove("hidden");
		$("#pullTextResult").classList.add("hidden");
		updateP2PStatus("pull", `${t("p2pDone")} · ${t("metaSize")}: ${humanSize(blob.size)}`, false, "");
		scheduleP2PDiagnostic(p2pState, "pull");
		toast(t("p2pDone"), "success");
	}

	function showP2PReceiveError(message) {
		if (p2pState) stopP2PStatusTimer(p2pState);
		$("#pullTextResult").classList.add("hidden");
		$("#pullFileResult").classList.add("hidden");
		$("#pullMeta").textContent = message;
		$("#pullResult").classList.remove("hidden");
		toast(message, "error");
	}

	// ── Pull ──────────────────────────────────────────────────

	function setSafeState(state) {
		const safe = $(".safe-visual");
		if (!safe) return;
		safe.classList.remove("safe-loading", "safe-open", "safe-error");
		if (!state) return;
		if (state === "error") {
			void safe.offsetWidth;
		}
		safe.classList.add("safe-" + state);
	}

	$("#pullBtn").addEventListener("click", async () => {
		const key = $("#pullKey").value.trim();
		if (!key) {
			setSafeState("error");
			toast(t("enterKeyWarn"), "error");
			return;
		}
		const pullMore = $(".pull-more");
		if (pullMore) pullMore.open = false;

		const btn = $("#pullBtn");
		btn.disabled = true;
		btn.textContent = t("pulling");
		setSafeState("loading");

		try {
			const headers = {
				...authHeaders(),
			};
			if ($("#deleteAfterPull").checked) {
				headers["Delete-After-Pull"] = "true";
			}

			if (!headers["Delete-After-Pull"]) {
				const jsonResp = await fetch(apiUrl("/pull/" + key), {
					headers: { ...headers, Accept: "application/json" },
				});

				const ct = jsonResp.headers.get("Content-Type") || "";

				if (ct.includes("application/json")) {
					const data = await jsonResp.json();
					if (data.code !== 0) {
						setSafeState("error");
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
						setSafeState("open");
						toast(t("pullOk"), "success");
						return;
					}
				}
			}

			const streamResp = await fetch(apiUrl("/pull/" + key), { headers });
			if (!streamResp.ok) {
				const errData = await streamResp.json().catch(() => null);
				setSafeState("error");
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
			const fileCount = streamResp.headers.get("Crossshare-File-Count");
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
			if (shareType) meta += ` · ${t("metaType")}: ${shareType}`;
			if (fileCount) meta += ` · ${t("metaFiles")}: ${fileCount}`;
			if (deleted) meta += ` · ${t("deleted")}`;
			$("#pullMeta").textContent = meta;
			$("#pullResult").classList.remove("hidden");
			setSafeState("open");
			toast(t("pullOk"), "success");
		} catch (e) {
			setSafeState("error");
			toast(t("reqFail") + ": " + e.message, "error");
		} finally {
			btn.disabled = false;
			btn.textContent = t("pull");
		}
	});

	$("#pullKey").addEventListener("input", () => setSafeState(""));

	$("#clearPull").addEventListener("click", () => {
		if (p2pState && p2pState.role === "receiver") {
			const state = p2pState;
			state.stopped = true;
			stopP2PTransport(state);
			p2pState = null;
		}

		$("#pullKey").value = "";
		$("#pullTextContent").textContent = "";
		$("#pullTextResult").classList.add("hidden");
		$("#pullFileName").textContent = "";
		$("#pullFileLink").removeAttribute("href");
		$("#pullFileLink").removeAttribute("download");
		$("#pullFileResult").classList.add("hidden");
		$("#pullMeta").textContent = "";
		$("#pullResult").classList.add("hidden");
		setSafeState("");
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

	// ── Countdown Timer ──────────────────────────────────────

	let countdownTimer = null;

	function stopCountdown() {
		if (countdownTimer) clearInterval(countdownTimer);
		countdownTimer = null;
		const el = $("#countdownDisplay");
		if (el) el.remove();
	}

	function startCountdown(expireAt) {
		stopCountdown();

		let el = $("#countdownDisplay");
		if (!el) {
			el = document.createElement("div");
			el.id = "countdownDisplay";
			el.className = "countdown";
			const resultMeta = $("#resultMeta");
			resultMeta.parentNode.insertBefore(el, resultMeta);
		}

		function update() {
			const remaining = expireAt - Math.floor(Date.now() / 1000);
			if (remaining <= 0) {
				el.textContent = t("expired");
				el.classList.add("countdown-expired");
				el.classList.remove("countdown-warning");
				clearInterval(countdownTimer);
				countdownTimer = null;
				return;
			}
			el.classList.remove("countdown-expired");
			if (remaining <= 60) {
				el.classList.add("countdown-warning");
			} else {
				el.classList.remove("countdown-warning");
			}
			el.textContent = `${t("expiresIn")}: ${humanDuration(remaining)}`;
		}

		update();
		countdownTimer = setInterval(update, 1000);
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
		if (e.key === "Escape") {
			closeQrModal();
			closeSettingsPanel();
			return;
		}
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
