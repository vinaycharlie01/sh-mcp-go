package kubernetes

import (
	"encoding/json"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// marshalCRD serialises a CRD to JSON for server-side apply.
func marshalCRD(crd *apiextv1.CustomResourceDefinition) ([]byte, error) {
	return json.Marshal(crd)
}
