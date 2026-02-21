package catalogdata

import "embed"

// EngineFS contains embedded engine asset YAML files from catalog/engines/.
//
//go:embed engines/*/*.yaml
var EngineFS embed.FS
