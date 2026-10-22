// Package data embeds the venue catalog in the API binary. Updating data
// means editing the JSON files and redeploying — there is no database.
package data

import "embed"

// FS holds venues.json and offerings.json at its root.
//
//go:embed *.json
var FS embed.FS
