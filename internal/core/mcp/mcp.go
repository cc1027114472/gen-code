package mcp

// Server describes an MCP endpoint known to runtime.
type Server struct {
	Name string
	URL  string
}

// Catalog tracks registered MCP servers.
type Catalog struct {
	Servers []Server
}
