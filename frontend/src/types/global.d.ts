// Ambient types for the Wails-injected globals. The Wails runtime attaches the
// generated Go bindings to `window.go` at startup; typing it here against the
// generated App module gives every `window.go.main.App.*` call its real
// signature (params + return types) without importing anything.
import type * as App from '../../wailsjs/go/main/App'

declare global {
    interface Window {
        go?: {
            main?: {
                App?: typeof App
            }
        }
    }
}

export {}
