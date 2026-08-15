package static

import "io/fs"

// Dist holds the embedded frontend build when compiled with `-tags dist`,
// otherwise nil (dev/API-only binaries serve no static files).
var Dist fs.FS