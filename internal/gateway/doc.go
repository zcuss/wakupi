// Package gateway turns Wakupi into a programmable WhatsApp gateway
// (n8n-style: receive events, send commands, react with rules).
//
// The gateway package is intentionally framework-free: it reads/writes a
// small YAML config from disk and exposes a Dispatcher that subscribes to
// WhatsApp events emitted by wa.Manager. It does not import Wails or any
// other UI layer, so it can be unit-tested headlessly.
package gateway
