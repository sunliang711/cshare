package web

import "embed"

//go:embed index.html style.css app.js qrcode-lite.js logo.png pen.webp
var StaticFiles embed.FS
