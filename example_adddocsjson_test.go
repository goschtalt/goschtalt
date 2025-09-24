// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt_test

import (
	"fmt"

	"github.com/goschtalt/goschtalt"
)

// ExampleAddDocsJSON demonstrates how to use AddDocsJSON to add documentation
// to your configuration from a JSON structure. The JSON format follows the
// doc.Object structure defined in the pkg/doc package.
//
// JSON Structure:
//
// The JSON must represent a valid documentation tree starting from the root.
// Each object in the tree has the following fields:
//
//   - "Name": string - The name of the field/object (empty string "" for root, or omitted)
//   - "Doc": string - The documentation text for this field (optional)
//   - "Tag": string - The struct tag for this field (optional)
//   - "Type": string - The type of the field (required, see types below)
//   - "Deprecated": bool - Whether this field is deprecated (optional, default false)
//   - "Optional": bool - Whether this field is optional (optional, default false)
//   - "Children": object - Map of child objects for complex types (optional)
//
// Valid Types:
//
//   - "<root>" - Root of the documentation tree
//   - "<map>" - Map/dictionary type
//   - "<struct>" - Struct type
//   - "<array>" - Array/slice type
//   - "<string>" - String type
//   - "<bool>" - Boolean type
//   - "<int>", "<int8>", "<int16>", "<int32>", "<int64>" - Integer types
//   - "<uint>", "<uint8>", "<uint16>", "<uint32>", "<uint64>", "<uintptr>" - Unsigned integer types
//   - "<float32>", "<float64>" - Floating point types
//   - "<complex64>", "<complex128>" - Complex number types
//
// Special Names for Children:
//
//   - "<array>" - Used for array element documentation
//   - "<key>" - Used for map key documentation
//   - "<value>" - Used for map value documentation
//   - "<embedded>" - Used for embedded struct documentation
//
// Root Object Requirements:
//
//   - Name must be empty string ""
//   - Type must be "<root>"
func ExampleAddDocsJSON() {
	// Define a configuration structure
	type DatabaseConfig struct {
		Host     string
		Port     int
		Username string
		Password string
		SSL      bool
	}

	type ServerConfig struct {
		Database DatabaseConfig
		Debug    bool
		Tags     []string
	}

	// Create documentation using JSON format
	docsJSON := `{
		"Type": "<root>",
		"Doc": "Server configuration settings",
		"Children": {
			"Database": {
				"Name": "Database",
				"Type": "<struct>",
				"Doc": "Database connection configuration",
				"Children": {
					"Host": {
						"Name": "Host",
						"Type": "<string>",
						"Doc": "database server hostname or IP address"
					},
					"Port": {
						"Name": "Port",
						"Type": "<int>",
						"Doc": "database server port number (typically 5432 for PostgreSQL)"
					},
					"Username": {
						"Name": "Username",
						"Type": "<string>",
						"Doc": "database username for authentication"
					},
					"Password": {
						"Name": "Password",
						"Type": "<string>",
						"Doc": "database password for authentication",
						"Deprecated": true
					},
					"SSL": {
						"Name": "SSL",
						"Type": "<bool>",
						"Doc": "Enable ssl/TLS connection to database"
					}
				}
			},
			"Debug": {
				"Name": "Debug",
				"Type": "<bool>",
				"Doc": "Enable debug logging and verbose output"
			},
			"Tags": {
				"Name": "Tags",
				"Type": "<array>",
				"Doc": "List of tags to apply to this server instance",
				"Children": {
					"<array>": {
						"Name": "<array>",
						"Type": "<string>",
						"Doc": "Individual tag name"
					}
				}
			}
		}
	}`

	// Create configuration with documentation
	cfg, err := goschtalt.New(
		goschtalt.AddValue("config", goschtalt.Root, ServerConfig{
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "admin",
				Password: "secret",
				SSL:      true,
			},
			Debug: false,
			Tags:  []string{"production", "primary"},
		}),
		// Add documentation from JSON
		goschtalt.AddDocsJSON([]byte(docsJSON)),
	)
	if err != nil {
		panic(err)
	}

	// Marshal the configuration with documentation
	output, err := cfg.Marshal(goschtalt.IncludeDocumentation())
	if err != nil {
		panic(err)
	}

	fmt.Print(string(output))

	// Output:
	// ---
	// # Database connection configuration
	// # type: <struct>
	// Database:
	//
	//   # database server hostname or IP address
	//   # type: <string>
	//   Host: localhost
	//
	//   # !!! DEPRECATED !!!
	//   # database password for authentication
	//   # type: <string>
	//   # !!! DEPRECATED !!!
	//   Password: secret
	//
	//   # database server port number (typically 5432 for PostgreSQL)
	//   # type: <int>
	//   Port: '5432'
	//
	//   # Enable ssl/TLS connection to database
	//   # type: <bool>
	//   SSL: 'true'
	//
	//   # database username for authentication
	//   # type: <string>
	//   Username: admin
	//
	// # Enable debug logging and verbose output
	// # type: <bool>
	// Debug: 'false'
	//
	// # List of tags to apply to this server instance
	// # type: array of <string>
	// Tags:
	//
	//   # Individual tag name
	//   # type: <string>
	//   - production
	//   - primary
}

// ExampleAddDocsJSON_arrayAndMapTypes demonstrates more complex documentation
// scenarios including arrays and maps with detailed type documentation.
func ExampleAddDocsJSON_arrayAndMapTypes() {
	type Config struct {
		Services map[string][]string
		Metrics  map[string]int
	}

	// Documentation for complex types
	docsJSON := `{
		"Type": "<root>",
		"Children": {
			"Services": {
				"Name": "Services",
				"Type": "<map>",
				"Doc": "Service endpoint mappings",
				"Children": {
					"<key>": {
						"Name": "<key>",
						"Type": "<string>",
						"Doc": "Service name identifier\n(e.g., 'api', 'database')"
					},
					"<value>": {
						"Name": "<value>",
						"Type": "<array>",
						"Doc": "List of endpoint URLs for this service\n(e.g., 'http://api.example.com/v1', 'http://api.example.com/v2')",
						"Children": {
							"<array>": {
								"Name": "<array>",
								"Type": "<string>",
								"Doc": "Individual endpoint URL"
							}
						}
					}
				}
			},
			"Metrics": {
				"Name": "Metrics",
				"Type": "<map>",
				"Doc": "Performance metrics and thresholds",
				"Children": {
					"<key>": {
						"Name": "<key>",
						"Type": "<string>",
						"Doc": "Metric name"
					},
					"<value>": {
						"Name": "<value>",
						"Type": "<int>",
						"Doc": "Metric threshold value"
					}
				}
			}
		}
	}`

	cfg, err := goschtalt.New(
		goschtalt.AddValue("config", goschtalt.Root, Config{
			Services: map[string][]string{
				"api":      {"http://api1.example.com", "http://api2.example.com"},
				"database": {"postgresql://db1.example.com:5432"},
			},
			Metrics: map[string]int{
				"max_connections": 100,
				"timeout_seconds": 30,
			},
		}),
		goschtalt.AddDocsJSON([]byte(docsJSON)),
	)
	if err != nil {
		panic(err)
	}

	output, err := cfg.Marshal(goschtalt.IncludeDocumentation())
	if err != nil {
		panic(err)
	}

	fmt.Print(string(output))

	// Output:
	// ---
	// # Performance metrics and thresholds
	// # type: map with key <string> -> value <int>
	// #   key(<string>) Metric name
	// #   value(<int>) Metric threshold value
	// Metrics:
	//   max_connections: '100'
	//   timeout_seconds: '30'
	//
	// # Service endpoint mappings
	// # type: map with key <string> -> value array of <string>
	// #   key(<string>) Service name identifier
	// #                 (e.g., 'api', 'database')
	// #   value(array of <string>) List of endpoint URLs for this service
	// #                            (e.g., 'http://api.example.com/v1', 'http://api.example.com/v2')
	// Services:
	//   api:
	//     - http://api1.example.com
	//     - http://api2.example.com
	//   database:
	//     - postgresql://db1.example.com:5432
}

// ExampleAddDocsJSON_multipleDocTrees demonstrates how to add multiple
// documentation trees that get merged together.
func ExampleAddDocsJSON_multipleDocTrees() {
	type Config struct {
		Server   map[string]any
		Database map[string]any
	}

	// First documentation tree for server configuration
	serverDocsJSON := `{
		"Type": "<root>",
		"Children": {
			"Server": {
				"Name": "Server",
				"Type": "<map>",
				"Doc": "Web server configuration settings",
				"Children": {
					"port": {
						"Name": "port",
						"Type": "<int>",
						"Doc": "HTTP server port number"
					},
					"host": {
						"Name": "host",
						"Type": "<string>",
						"Doc": "Server bind address"
					}
				}
			}
		}
	}`

	// Second documentation tree for database configuration
	databaseDocsJSON := `{
		"Type": "<root>",
		"Children": {
			"Database": {
				"Name": "Database",
				"Type": "<map>",
				"Doc": "Database connection settings",
				"Children": {
					"driver": {
						"Name": "driver",
						"Type": "<string>",
						"Doc": "Database driver name (e.g., 'postgres', 'mysql')"
					},
					"dsn": {
						"Name": "dsn",
						"Type": "<string>",
						"Doc": "Data source name connection string"
					}
				}
			}
		}
	}`

	cfg, err := goschtalt.New(
		goschtalt.AddValue("config", goschtalt.Root, Config{
			Server: map[string]any{
				"port": 8080,
				"host": "0.0.0.0",
			},
			Database: map[string]any{
				"driver": "postgres",
				"dsn":    "postgresql://user:pass@localhost/db",
			},
		}),
		// Add multiple documentation trees - they will be merged
		goschtalt.AddDocsJSON([]byte(serverDocsJSON)),
		goschtalt.AddDocsJSON([]byte(databaseDocsJSON)),
	)
	if err != nil {
		panic(err)
	}

	output, err := cfg.Marshal(goschtalt.IncludeDocumentation())
	if err != nil {
		panic(err)
	}

	fmt.Print(string(output))

	// Output:
	// ---
	// # Database connection settings
	// # type: <map>
	// Database:
	//
	//   # Database driver name (e.g., 'postgres', 'mysql')
	//   # type: <string>
	//   driver: postgres
	//
	//   # Data source name connection string
	//   # type: <string>
	//   dsn: 'postgresql://user:pass@localhost/db'
	//
	// # Web server configuration settings
	// # type: <map>
	// Server:
	//
	//   # Server bind address
	//   # type: <string>
	//   host: '0.0.0.0'
	//
	//   # HTTP server port number
	//   # type: <int>
	//   port: '8080'
}
