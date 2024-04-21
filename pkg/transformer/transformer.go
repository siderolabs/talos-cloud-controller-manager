// Package transformer provides a mechanism to transform node specification based on node transformation rules.
package transformer

import (
	"bytes"
	"fmt"
	"html/template"
	"reflect"
	"slices"
	"strings"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/nodeselector"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// NodeTerm represents expressions and fields to transform node metadata.
type NodeTerm struct {
	Name             string                          `yaml:"name,omitempty"`
	NodeSelector     []nodeselector.NodeSelectorTerm `yaml:"nodeSelector,omitempty"`
	Annotations      map[string]string               `yaml:"annotations,omitempty"`
	Labels           map[string]string               `yaml:"labels,omitempty"`
	PlatformMetadata map[string]string               `yaml:"platformMetadata,omitempty"`
}

// NodeSpec represents the transformed node specifcations.
type NodeSpec struct {
	Annotations map[string]string
	Labels      map[string]string
}

var prohibitedPlatformMetadataKeys = []string{"hostname", "platform"}

// TransformNode transforms the node metadata based on the node transformation rules.
func TransformNode(terms []NodeTerm, platformMetadata *runtime.PlatformMetadataSpec) (*NodeSpec, error) {
	if len(terms) == 0 {
		return nil, nil
	}

	metadata := metadataFromStruct(platformMetadata)

	for _, term := range terms {
		match, err := nodeselector.Match(term.NodeSelector, metadata)
		if err != nil {
			return nil, err
		}

		if match {
			node := &NodeSpec{
				Annotations: make(map[string]string),
				Labels:      make(map[string]string),
			}

			if term.Annotations != nil {
				for k, v := range term.Annotations {
					t, err := executeTemplate(v, platformMetadata)
					if err != nil {
						return nil, fmt.Errorf("failed to transformer annotation '%q': %w", k, err)
					}

					node.Annotations[k] = t
				}
			}

			if term.Labels != nil {
				for k, v := range term.Labels {
					t, err := executeTemplate(v, platformMetadata)
					if err != nil {
						return nil, fmt.Errorf("failed to transformer label '%s': %w", k, err)
					}

					node.Labels[k] = t
				}
			}

			if term.PlatformMetadata != nil {
				p := reflect.ValueOf(platformMetadata)
				ps := p.Elem()

				for k, v := range term.PlatformMetadata {
					if slices.Contains(prohibitedPlatformMetadataKeys, strings.ToLower(k)) {
						continue
					}

					t, err := executeTemplate(v, platformMetadata)
					if err != nil {
						return nil, fmt.Errorf("failed to transformer platform metadata '%s': %w", k, err)
					}

					f := ps.FieldByNameFunc(func(fieldName string) bool {
						return strings.EqualFold(fieldName, k)
					})

					if f.IsValid() {
						switch f.Kind() { //nolint:exhaustive
						case reflect.Bool:
							f.SetBool(t == "true")
						case reflect.String:
							f.SetString(strings.TrimSpace(t))
						default:
							return nil, fmt.Errorf("unsupported platform metadata field %s", k)
						}
					}
				}
			}

			return node, nil
		}
	}

	return nil, nil
}

func executeTemplate(tmpl string, data interface{}) (string, error) {
	t, err := template.New("transformer").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %q: %w", tmpl, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func metadataFromStruct(in *runtime.PlatformMetadataSpec) map[string]string {
	if in == nil {
		return nil
	}

	metadata := make(map[string]string)

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			// skip unexported fields
			if !v.Field(i).CanInterface() {
				continue
			}

			tag := v.Type().Field(i).Tag.Get("yaml")
			if tag == "" {
				continue
			}

			fieldName := strings.ToLower(strings.Split(tag, ",")[0])

			reflectedValue := reflect.Indirect(v.Field(i))
			if reflectedValue.IsValid() {
				switch reflectedValue.Kind() { //nolint:exhaustive
				case reflect.Bool:
					metadata[fieldName] = fmt.Sprintf("%t", reflectedValue.Bool())
				case reflect.String:
					v := reflectedValue.String()
					if v != "" {
						metadata[fieldName] = v
					}
				}
			}
		}
	}

	return metadata
}
