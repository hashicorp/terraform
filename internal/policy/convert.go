// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

// ResourceAttributesToProto serializes a policy value, including its redacted paths,
// for resource callback responses.
func ResourceAttributesToProto(value PolicyValue) (*proto.ResourceAttributes, error) {
	return resourceAttributesToProto(value)
}

func resourceAttributesToProto(value PolicyValue) (*proto.ResourceAttributes, error) {
	raw, err := msgpack.Marshal(value.Raw, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("error serializing raw value: %w", err)
	}

	redactedPaths, err := pathsToProto(value.RedactedPaths)
	if err != nil {
		return nil, fmt.Errorf("error serializing redacted paths: %w", err)
	}

	return &proto.ResourceAttributes{
		Raw:           raw,
		RedactedPaths: redactedPaths,
	}, nil
}

func pathToProto(path cty.Path) (*proto.AttributePath, error) {
	steps := make([]*proto.AttributePath_Step, 0, len(path))
	for _, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			steps = append(steps, &proto.AttributePath_Step{
				Selector: &proto.AttributePath_Step_AttributeName{AttributeName: step.Name},
			})
		case cty.IndexStep:
			key := step.Key
			switch key.Type() {
			case cty.String:
				steps = append(steps, &proto.AttributePath_Step{
					Selector: &proto.AttributePath_Step_ElementKeyString{ElementKeyString: key.AsString()},
				})
			case cty.Number:
				v, _ := key.AsBigFloat().Int64()
				steps = append(steps, &proto.AttributePath_Step{
					Selector: &proto.AttributePath_Step_ElementKeyInt{ElementKeyInt: int64(v)},
				})
			default:
				return nil, fmt.Errorf("unsupported cty path step type %T", step)
			}
		default:
			return nil, fmt.Errorf("unsupported cty path step type %T", step)
		}
	}
	return &proto.AttributePath{Steps: steps}, nil
}

// configFromProto prepares both schema-validation definitions and evaluation
// trees before Setup publishes the configuration. Nil capabilities are a valid
// empty configuration, rather than failed initialization.
func configFromProto(capabilities *proto.PolicySetupResponse_ServerCapabilities) (*Config, error) {
	if capabilities == nil {
		return &Config{}, nil
	}
	definitions, err := relationshipsFromProto(capabilities.Relationships)
	if err != nil {
		return nil, err
	}
	trees := make(map[string][]Relationship, len(capabilities.ResourceRelationships))
	for _, tree := range capabilities.ResourceRelationships {
		if tree == nil || tree.ResourceType == "" {
			return nil, fmt.Errorf("relationship usage tree has no resource type")
		}
		if _, exists := trees[tree.ResourceType]; exists {
			return nil, fmt.Errorf("duplicate relationship usage tree for resource type %q", tree.ResourceType)
		}
		relationships, err := relationshipReferencesFromProto(definitions, tree.ResourceType, tree.References)
		if err != nil {
			return nil, fmt.Errorf("resource type %q: %w", tree.ResourceType, err)
		}
		trees[tree.ResourceType] = relationships
	}
	return &Config{Definitions: definitions, Relationships: trees}, nil
}

// relationshipsFromProto decodes each shared definition once. References use
// these read-only matching rules while retaining their own nested usage trees.
func relationshipsFromProto(relationships []*proto.Relationship) (map[string]Relationship, error) {
	ret := make(map[string]Relationship, len(relationships))
	for _, relationship := range relationships {
		if relationship == nil || relationship.Name == "" {
			return nil, fmt.Errorf("relationship definition has no name")
		}
		if _, exists := ret[relationship.Name]; exists {
			return nil, fmt.Errorf("duplicate relationship definition %q", relationship.Name)
		}
		if relationship.KeyType == "" || relationship.ValueType == "" {
			return nil, fmt.Errorf("relationship %q: endpoint types must not be empty", relationship.Name)
		}
		matches := make([]Match, 0, len(relationship.Matches))
		for _, match := range relationship.Matches {
			if match == nil {
				return nil, fmt.Errorf("relationship %q: missing match", relationship.Name)
			}
			keys := make(Match, 0, len(match.Keys))
			for _, key := range match.Keys {
				if key == nil {
					return nil, fmt.Errorf("relationship %q: missing match key", relationship.Name)
				}
				keyPath, err := relationshipPathFromProto(key.KeyPath)
				if err != nil {
					return nil, fmt.Errorf("relationship %q: invalid key path: %w", relationship.Name, err)
				}
				valuePath, err := relationshipPathFromProto(key.ValuePath)
				if err != nil {
					return nil, fmt.Errorf("relationship %q: invalid value path: %w", relationship.Name, err)
				}
				keys = append(keys, RelationshipKey{KeyPath: keyPath, ValuePath: valuePath})
			}
			matches = append(matches, keys)
		}

		ret[relationship.Name] = Relationship{
			Name:      relationship.Name,
			KeyType:   relationship.KeyType,
			ValueType: relationship.ValueType,
			Matches:   matches,
		}
	}
	return ret, nil
}

func relationshipPathFromProto(path *proto.AttributePath) (cty.Path, error) {
	if path == nil {
		return nil, fmt.Errorf("missing attribute path")
	}
	for _, step := range path.Steps {
		if step == nil || step.Selector == nil {
			return nil, fmt.Errorf("missing attribute path selector")
		}
	}
	return path.ToCtyPath()
}

// relationshipReferencesFromProto resolves a finite usage tree, checking that
// each relationship connects to the resource type at that level. Repeated names
// are valid: recursion follows the explicit reference tree, not the definitions.
func relationshipReferencesFromProto(definitions map[string]Relationship, resourceType string, references []*proto.ResourceRelationships_Reference) ([]Relationship, error) {
	ret := make([]Relationship, 0, len(references))
	for _, reference := range references {
		if reference == nil {
			return nil, fmt.Errorf("missing relationship reference")
		}
		relationship, exists := definitions[reference.Relationship]
		if !exists {
			return nil, fmt.Errorf("undefined relationship %q", reference.Relationship)
		}
		var otherType string
		switch resourceType {
		case relationship.KeyType:
			otherType = relationship.ValueType
		case relationship.ValueType:
			otherType = relationship.KeyType
		default:
			return nil, fmt.Errorf("relationship %q does not connect to resource type %q", relationship.Name, resourceType)
		}
		nested, err := relationshipReferencesFromProto(definitions, otherType, reference.References)
		if err != nil {
			return nil, err
		}
		relationship.Relationships = nested
		ret = append(ret, relationship)
	}
	return ret, nil
}

func relationsToProto(relations []Relation) []*proto.Relation {
	ret := make([]*proto.Relation, 0, len(relations))
	for _, relation := range relations {
		resources := make(map[string]*proto.RelatedResource, len(relation.Resources))
		for identifier, nested := range relation.Resources {
			resources[identifier] = &proto.RelatedResource{
				Relationships: relationsToProto(nested),
			}
		}
		ret = append(ret, &proto.Relation{
			Relationship: relation.Relationship,
			Resources:    resources,
			Complete:     relation.Complete,
		})
	}
	return ret
}

func pathsToProto(paths []cty.Path) ([]*proto.AttributePath, error) {
	ret := make([]*proto.AttributePath, 0, len(paths))
	for _, path := range paths {
		protoPath, err := pathToProto(path)
		if err != nil {
			return nil, err
		}
		ret = append(ret, protoPath)
	}
	return ret, nil
}
