/*
Copyright 2019 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transformer

import (
	"reflect"

	"github.com/carbonrelay/konjure/internal/berglas"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	h *resmap.PluginHelpers

	GeneratorOptions *types.GeneratorOptions `json:"generatorOptions,omitempty"`
	GenerateSecrets  bool                    `json:"-"` // TODO Change back to "generateSecrets,omitempty
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	opts := p.GeneratorOptions
	if opts == nil && p.GenerateSecrets {
		opts = &types.GeneratorOptions{}
	} else if !p.GenerateSecrets {
		opts = nil
	}

	// Create a new mutator
	mutator := berglas.NewMutator(p.h, opts)
	for _, r := range m.Resources() {
		// Mutate using the appropriate API struct
		if err := mutateResourceAs(mutator, r); err != nil {
			return err
		}

		// Check if there were any new secrets that need to be added
		if err := mutator.FlushSecrets(m); err != nil {
			return err
		}
	}

	return nil
}

// Performs the Berglas mutation on a Kustomize resource
func mutateResourceAs(m *berglas.Mutator, r *resource.Resource) error {
	// Create a new typed object
	obj, err := scheme.Scheme.New(toSchemaGvk(r.GetGvk()))
	if err != nil {
		return nil // This is ignorable
	}

	// Marshal the unstructured resource to JSON and unmarshal it back into the typed structure
	if data, err := r.MarshalJSON(); err != nil {
		return err
	} else if err := json.Unmarshal(data, obj); err != nil {
		return err
	}

	// Reflectively get a pointer to the PodTemplateSpec
	template := podTemplate(obj)
	if template == nil {
		return nil
	}

	// Mutate the PodTemplateSpec and if there were changes, reverse the marshalling process back into the resource
	if didMutate, err := m.Mutate(template); err != nil {
		return err
	} else if didMutate {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		return r.UnmarshalJSON(data)
	}
	return nil
}

func toSchemaGvk(x resid.Gvk) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   x.Group,
		Version: x.Version,
		Kind:    x.Kind,
	}
}

func podTemplate(obj runtime.Object) *corev1.PodTemplateSpec {
	v := reflect.ValueOf(obj)
	if !v.CanInterface() {
		return nil
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}

	v = v.FieldByName("Spec")
	if v.Kind() != reflect.Struct {
		return nil
	}

	v = v.FieldByName("Template")
	if !v.CanAddr() {
		return nil
	}

	v = v.Addr()
	if !v.CanInterface() {
		return nil
	}

	if t, ok := v.Interface().(*corev1.PodTemplateSpec); ok {
		return t
	}

	return nil
}
