package browser

// simpleDescription returns the tool description for simple level (~305 tokens).
// Covers 11 core actions sufficient for most browser automation tasks.
func simpleDescription() string {
	return `Control a browser to navigate web pages, take accessibility snapshots, and interact with elements.

Actions:
- status: Get browser status
- start: Launch browser
- stop: Close browser
- tabs: List open tabs
- open: Open a new tab (requires targetUrl)
- close: Close a tab (requires targetId)
- snapshot: Get page accessibility tree with element refs (use targetId, maxChars, interactive, compact, depth)
- screenshot: Capture page screenshot (use targetId, fullPage)
- navigate: Navigate tab to URL (requires targetId, targetUrl)
- console: Get browser console messages (requires targetId)
- act: Interact with elements (requires request object with kind, ref, etc.)

Act kinds: click, type, press, hover, wait, evaluate
- click: Click element (request: {kind:"click", ref:"e1"})
- type: Type text (request: {kind:"type", ref:"e1", text:"hello"})
- press: Press key (request: {kind:"press", key:"Enter"})
- hover: Hover element (request: {kind:"hover", ref:"e1"})
- wait: Wait for condition (request: {kind:"wait", timeMs:1000} or {kind:"wait", text:"loaded"})
- evaluate: Run JavaScript (request: {kind:"evaluate", fn:"document.title"})`
}

// simpleParameters returns the JSON Schema parameters for simple level.
// 10 top-level params + request object.
func simpleParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"status", "start", "stop", "tabs", "open", "close",
					"snapshot", "screenshot", "navigate", "console", "act",
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
			"fullPage": map[string]any{
				"type":        "boolean",
				"description": "Capture full page screenshot",
			},
			"timeoutMs": map[string]any{
				"type":        "number",
				"description": "Timeout in milliseconds for actions",
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
