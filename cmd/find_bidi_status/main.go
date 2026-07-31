package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

func main() {
	cacheDir := os.Getenv("HOME") + "/.cache/librarian"
	absSpecSource := filepath.Join(cacheDir, "github.com/googleapis/googleapis@b8486a2f44f15dc578a9dc1e17b144253079d5c1")
	showcaseSpecSource := filepath.Join(cacheDir, "github.com/googleapis/gapic-showcase@3fd9cb2f682d5f8263d913eaba8b78e045acc4d2/schema")

	srcs := &sources.Sources{
		Googleapis: absSpecSource,
		Showcase:   showcaseSpecSource,
	}

	configPath := "/usr/local/google/home/suzmue/generate-streaming-work/google-cloud-rust-google-rpc-status-support/librarian.yaml"
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Println("Scanning libraries for Bidi-Streaming RPCs returning google.rpc.Status...\n")

	for _, lib := range cfg.Libraries {
		if lib.SkipGenerate {
			continue
		}

		for _, apiObj := range lib.APIs {
			specSource := absSpecSource
			root := "googleapis"
			if len(lib.Roots) > 0 && lib.Roots[0] == "showcase" {
				specSource = showcaseSpecSource
				root = "showcase"
			}
			apiPath := filepath.Join(specSource, apiObj.Path)

			modelConfig := &parser.ModelConfig{
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: apiPath,
				Source:              sources.NewSourceConfig(srcs, []string{root}),
			}

			model, err := parser.CreateModel(modelConfig)
			if err != nil {
				continue
			}

			for _, s := range model.Services {
				for _, method := range s.Methods {
					if method.ClientSideStreaming && method.ServerSideStreaming {
						// Check if input or output message references google.rpc.Status
						hasStatus := false
						if method.InputType != nil && referencesStatus(method.InputType, model, make(map[string]bool)) {
							hasStatus = true
						}
						if method.OutputType != nil && referencesStatus(method.OutputType, model, make(map[string]bool)) {
							hasStatus = true
						}
						if hasStatus {
							fmt.Printf("Library: %s\n", lib.Name)
							fmt.Printf("  Service: %s\n", s.ID)
							fmt.Printf("  RPC: %s\n", method.Name)
							fmt.Printf("  Input: %s\n", method.InputType.ID)
							fmt.Printf("  Output: %s\n\n", method.OutputType.ID)
						}
					}
				}
			}
		}
	}
}

func referencesStatus(msg *api.Message, model *api.API, visited map[string]bool) bool {
	if msg == nil || visited[msg.ID] {
		return false
	}
	visited[msg.ID] = true

	if strings.TrimPrefix(msg.ID, ".") == "google.rpc.Status" {
		return true
	}

	for _, f := range msg.Fields {
		if strings.TrimPrefix(f.TypezID, ".") == "google.rpc.Status" {
			return true
		}
		if f.Typez == api.TypezMessage && f.TypezID != "" {
			childMsg := model.Message(f.TypezID)
			if childMsg != nil && referencesStatus(childMsg, model, visited) {
				return true
			}
		}
	}
	for _, o := range msg.OneOfs {
		for _, f := range o.Fields {
			if strings.TrimPrefix(f.TypezID, ".") == "google.rpc.Status" {
				return true
			}
			if f.Typez == api.TypezMessage && f.TypezID != "" {
				childMsg := model.Message(f.TypezID)
				if childMsg != nil && referencesStatus(childMsg, model, visited) {
					return true
				}
			}
		}
	}
	return false
}
