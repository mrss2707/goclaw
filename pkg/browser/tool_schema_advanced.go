package browser

// advancedDescription returns the tool description for advanced level (~400 tokens).
// Full 35 actions. Workflow guidance moved to system prompt to save tokens here.
func advancedDescription() string {
	return `Control a real Chrome/Chromium browser to navigate web pages, take accessibility snapshots, and interact with elements.

Actions:
- status: Get browser status (shows if running, headless mode, open tabs)
- start: Launch the browser engine
- stop: Close the browser engine and all tabs
- tabs: List open tabs
- open: Open a new tab (requires targetUrl; optional profile, width/height)
- close: Close a tab (requires targetId)
- focusTab: Activate/focus a tab (requires targetId)
- snapshot: Get page accessibility tree with element refs (use targetId, maxChars, interactive, compact, depth, includeFrames, frameId)
- frames: List all frames/iframes in the page (use targetId)
- screenshot: Capture page screenshot (use targetId, fullPage)
- navigate: Navigate tab to URL (requires targetId, targetUrl)
- console: Get browser console messages (requires targetId)
- act: Interact with elements (requires request object with kind, ref, etc.)
- attach: Connect to existing browser (requires cdpUrl)
- getCookies: Get page cookies (use targetId)
- setCookie: Set a cookie (requires cookie object with name, value, domain, etc.)
- clearCookies: Clear all cookies (use targetId)
- getStorage: Get localStorage/sessionStorage items (use targetId, storageKind: "local"|"session")
- setStorage: Set storage item (requires storageKey, storageValue; use storageKind)
- clearStorage: Clear storage (use targetId, storageKind)
- profiles: List saved browser profiles
- deleteProfile: Delete a browser profile (requires profile name)
- errors: Get captured JavaScript exceptions (use targetId)
- emulate: Set device/viewport emulation (use targetId, userAgent, width, height, scale, isMobile, hasTouch, landscape)
- pdf: Generate PDF from page (use targetId, landscape)
- setHeaders: Set extra HTTP headers (requires headers object, use targetId)
- setOffline: Enable/disable offline mode (use targetId, offline)
- startScreencast: Start streaming JPEG frames (use targetId, fps, quality)
- stopScreencast: Stop screencast streaming (use targetId)
- extension.list: List registered extensions
- audit.list: List browser audit log entries (optional auditAction, auditLimit)
- storage.purge: Purge a browser profile session (requires profile)
- storage.cleanup: Remove old profiles (requires maxAge in hours)
- liveview.create: Create a shareable public link for a browser tab (requires targetId; optional mode: view/takeover). Only use when user explicitly asks to share.

Act kinds: click, type, press, hover, wait, evaluate
- click: Click element (request: {kind:"click", ref:"e1"})
- type: Type text (request: {kind:"type", ref:"e1", text:"hello"})
- press: Press key (request: {kind:"press", key:"Enter"})
- hover: Hover element (request: {kind:"hover", ref:"e1"})
- wait: Wait for condition (request: {kind:"wait", timeMs:1000} or {kind:"wait", text:"loaded"})
- evaluate: Run JavaScript (request: {kind:"evaluate", fn:"document.title"})`
}

// advancedParameters returns the JSON Schema parameters for advanced level.
// Full 35 actions with all parameter options.
func advancedParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"status", "start", "stop", "tabs", "open", "close", "focusTab",
					"snapshot", "screenshot", "navigate", "console", "act", "frames",
					"attach", "getCookies", "setCookie", "clearCookies",
					"getStorage", "setStorage", "clearStorage",
					"profiles", "deleteProfile", "errors",
					"emulate", "pdf", "setHeaders", "setOffline",
					"startScreencast", "stopScreencast",
					"extension.list",
					"audit.list",
					"storage.purge", "storage.cleanup",
					"liveview.create",
				},
				"description": "The browser action to perform",
			},
			"targetUrl": map[string]any{
				"type":        "string",
				"description": "URL for open/navigate actions",
			},
			"targetId": map[string]any{
				"type":        "string",
				"description": "Tab target ID (omit for current tab)",
			},
			"cdpUrl": map[string]any{
				"type":        "string",
				"description": "CDP WebSocket URL for attach action",
			},
			"profile": map[string]any{
				"type":        "string",
				"description": "Browser profile name for open/deleteProfile actions",
			},
			"maxChars": map[string]any{
				"type":        "number",
				"description": "Max characters for snapshot (default 8000)",
			},
			"interactive": map[string]any{
				"type":        "boolean",
				"description": "Only show interactive elements in snapshot",
			},
			"compact": map[string]any{
				"type":        "boolean",
				"description": "Remove empty structural elements from snapshot",
			},
			"depth": map[string]any{
				"type":        "number",
				"description": "Max depth for snapshot tree",
			},
			"includeFrames": map[string]any{
				"type":        "boolean",
				"description": "Include child iframes in snapshot (opt-in, default false)",
			},
			"frameId": map[string]any{
				"type":        "string",
				"description": "Snapshot a specific frame by frame ID (from frames action)",
			},
			"fullPage": map[string]any{
				"type":        "boolean",
				"description": "Capture full page screenshot",
			},
			"timeoutMs": map[string]any{
				"type":        "number",
				"description": "Timeout in milliseconds for actions",
			},
			"cookie": map[string]any{
				"type":        "object",
				"description": "Cookie object for setCookie (name, value, domain, path, secure, httpOnly, sameSite, expires, url)",
			},
			"storageKind": map[string]any{
				"type":        "string",
				"enum":        []string{"local", "session"},
				"description": "Storage type: 'local' (default) or 'session'",
			},
			"storageKey": map[string]any{
				"type":        "string",
				"description": "Key for setStorage action",
			},
			"storageValue": map[string]any{
				"type":        "string",
				"description": "Value for setStorage action",
			},
			"userAgent": map[string]any{
				"type":        "string",
				"description": "User agent string for emulate action",
			},
			"width": map[string]any{
				"type":        "number",
				"description": "Viewport width for open (custom viewport) or emulate action",
			},
			"height": map[string]any{
				"type":        "number",
				"description": "Viewport height for open (custom viewport) or emulate action",
			},
			"scale": map[string]any{
				"type":        "number",
				"description": "Device scale factor for emulate action (default 1)",
			},
			"isMobile": map[string]any{
				"type":        "boolean",
				"description": "Enable mobile emulation",
			},
			"hasTouch": map[string]any{
				"type":        "boolean",
				"description": "Enable touch emulation",
			},
			"landscape": map[string]any{
				"type":        "boolean",
				"description": "Landscape orientation for emulate/pdf actions",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "HTTP headers object for setHeaders action",
			},
			"offline": map[string]any{
				"type":        "boolean",
				"description": "Enable offline mode for setOffline action",
			},
			"fps": map[string]any{
				"type":        "number",
				"description": "Frames per second for startScreencast (default 10)",
			},
			"quality": map[string]any{
				"type":        "number",
				"description": "JPEG quality for startScreencast (default 80)",
			},
			"auditAction": map[string]any{
				"type":        "string",
				"description": "Filter audit log by action name",
			},
			"auditLimit": map[string]any{
				"type":        "number",
				"description": "Max entries for audit.list (default 50)",
			},
			"maxAge": map[string]any{
				"type":        "number",
				"description": "Max age in hours for storage.cleanup",
			},
			"mode": map[string]any{
				"type":        "string",
				"enum":        []string{"view", "takeover"},
				"description": "LiveView mode: view (default) or takeover",
			},
			"expiresMinutes": map[string]any{
				"type":        "number",
				"description": "Token expiry in minutes for liveview.create (default 60, max 1440)",
			},
			"request": map[string]any{
				"type":        "object",
				"description": "Action request for 'act' command",
				"properties": map[string]any{
					"kind": map[string]any{
						"type":        "string",
						"enum":        []string{"click", "type", "press", "hover", "wait", "evaluate"},
						"description": "The interaction kind",
					},
					"ref": map[string]any{
						"type":        "string",
						"description": "Element ref from snapshot (e.g. e1, e2)",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "Text to type",
					},
					"key": map[string]any{
						"type":        "string",
						"description": "Key to press (e.g. Enter, Tab, Escape)",
					},
					"submit": map[string]any{
						"type":        "boolean",
						"description": "Press Enter after typing",
					},
					"fn": map[string]any{
						"type":        "string",
						"description": "JavaScript to evaluate",
					},
					"timeMs": map[string]any{
						"type":        "number",
						"description": "Wait time in milliseconds",
					},
				},
			},
		},
		"required": []string{"action"},
	}
}
