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

	"k8s.io/apimachinery/pkg/util/validation"
)

// NodeTerm represents expressions and fields to transform node metadata.
type NodeTerm struct {
	Name             string                          `yaml:"name,omitempty"`
	NodeSelector     []nodeselector.NodeSelectorTerm `yaml:"nodeSelector,omitempty"`
	Annotations      map[string]string               `yaml:"annotations,omitempty"`
	Labels           map[string]string               `yaml:"labels,omitempty"`
	PlatformMetadata map[string]string               `yaml:"platformMetadata,omitempty"`
	Features         NodeFeaturesFlagSpec            `yaml:"features,omitempty"`
}

// NodeSpec represents the transformed node specifcations.
type NodeSpec struct {
	Annotations map[string]string
	Labels      map[string]string
	Features    NodeFeaturesFlagSpec
}

// NodeFeaturesFlagSpec represents the node features flags.
type NodeFeaturesFlagSpec struct {
	// PublicIPDiscovery try to find public IP on the node
	PublicIPDiscovery bool `yaml:"publicIPDiscovery,omitempty"`
}

var prohibitedPlatformMetadataKeys = []string{"hostname", "platform"}

// TransformNode transforms the node metadata based on the node transformation rules.
//
//nolint:gocyclo,cyclop
func TransformNode(terms []NodeTerm, platformMetadata *runtime.PlatformMetadataSpec) (*NodeSpec, error) {
	if len(terms) == 0 {
		return nil, nil
	}

	node := &NodeSpec{
		Annotations: make(map[string]string),
		Labels:      make(map[string]string),
	}

	metadata := metadataFromStruct(platformMetadata)

	for _, term := range terms {
		match, err := nodeselector.Match(term.NodeSelector, metadata)
		if err != nil {
			return nil, err
		}

		if match {
			if term.Annotations != nil {
				for k, v := range term.Annotations {
					t, err := executeTemplate(v, platformMetadata)
					if err != nil {
						return nil, fmt.Errorf("failed to transformer annotation '%q': %w", k, err)
					}

					if errs := validation.IsQualifiedName(k); len(errs) != 0 {
						return nil, fmt.Errorf("invalid annotation name %q: %v", k, errs)
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

					if errs := validation.IsQualifiedName(k); len(errs) != 0 {
						return nil, fmt.Errorf("invalid label name %q: %v", k, errs)
					}

					if errs := validation.IsValidLabelValue(t); len(errs) != 0 {
						return nil, fmt.Errorf("invalid label value %q: %v", t, errs)
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
		}
	}

	return node, nil
}

func executeTemplate(tmpl string, data interface{}) (string, error) {
	t, err := template.New("transformer").Funcs(GenericFuncMap()).Parse(tmpl)
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
