/*
Copyright 2018 The Kubernetes Authors.

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

package webhook

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var genericFn = func(desired runtime.Object) error {
	return nil
}

func batchCreateOrUpdate(c client.Client, objs ...runtime.Object) error {
	for _, obj := range objs {
		// TODO: retry mutiple times with backoff if necessary.
		_, err := controllerutil.CreateOrUpdate(context.Background(), c, obj, genericFn)
		if err != nil {
			return err
		}
	}
	return nil
}
