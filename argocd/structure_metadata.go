package argocd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func expandMetadata(d *schema.ResourceData) (
	meta meta.ObjectMeta,
	diags diag.Diagnostics,
) {
	m := d.Get("metadata.0").(map[string]interface{})

	if v, ok := m["annotations"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Annotations = expandStringMap(v)
	}
	if v, ok := m["finalizers"]; ok {
		meta.Finalizers = expandStringList(v.(*schema.Set).List())

		if _, err := validateFinalizers(meta.Finalizers, "finalizers"); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Finalizers are invalid",
				Detail:   fmt.Errorf("finalizers invalid: %s", err).Error(),
			})
		}
	}
	if v, ok := m["labels"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Labels = expandStringMap(v)
	}
	if v, ok := m["name"]; ok {
		meta.Name = v.(string)
	}
	if v, ok := m["namespace"]; ok {
		meta.Namespace = v.(string)
	}
	return meta, diags
}

func flattenMetadata(meta meta.ObjectMeta, d *schema.ResourceData) []interface{} {
	m := map[string]interface{}{
		"generation":       meta.Generation,
		"name":             meta.Name,
		"namespace":        meta.Namespace,
		"resource_version": meta.ResourceVersion,
		"uid":              fmt.Sprintf("%v", meta.UID),
		"finalizers":       meta.Finalizers,
	}

	annotations := d.Get("metadata.0.annotations").(map[string]interface{})
	m["annotations"] = metadataRemoveInternalKeys(meta.Annotations, annotations)
	labels := d.Get("metadata.0.labels").(map[string]interface{})
	m["labels"] = metadataRemoveInternalKeys(meta.Labels, labels)

	return []interface{}{m}
}

func metadataRemoveInternalKeys(m map[string]string, d map[string]interface{}) map[string]string {
	for k := range m {
		if metadataIsInternalKey(k) && !isKeyInMap(k, d) {
			delete(m, k)
		}
	}
	return m
}

func metadataIsInternalKey(annotationKey string) bool {
	u, err := url.Parse("//" + annotationKey)
	if err == nil && strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}
	if err == nil && annotationKey == "notified.notifications.argoproj.io" {
		return true
	}
	return false
}
