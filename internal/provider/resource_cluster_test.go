package provider

import (
	"fmt"
	"regexp"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func TestAccArgoCDCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - bare minimum config
			{
				Config: `
resource "argocd_cluster" "this" {
	server     = "https://kubernetes.default.svc.cluster.local"

	config = {
		bearer_token = "abcdef.0123456789abcdef" # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.

		tls_client_config = {
			insecure = true
		}
	}
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_cluster.this", "config.tls_client_config.insecure", "true"),
					resource.TestCheckNoResourceAttr("argocd_cluster.this", "annotations"),
					resource.TestCheckNoResourceAttr("argocd_cluster.this", "labels"),
					resource.TestCheckNoResourceAttr("argocd_cluster.this", "name"),
					resource.TestCheckNoResourceAttr("argocd_cluster.this", "namespaces"),
					resource.TestCheckNoResourceAttr("argocd_cluster.this", "project"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "server", "https://kubernetes.default.svc.cluster.local"),
					resource.TestCheckNoResourceAttr("argocd_cluster.this", "shard"),
					// resource.TestCheckResourceAttr("argocd_cluster.this", "info.applications_count", "0"),
					// resource.TestCheckResourceAttr("argocd_cluster.this", "info.connection_state.status", "Successful"),
					// resource.TestCheckResourceAttr("argocd_cluster.this", "info.server_version", "1.24"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "argocd_cluster.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.bearer_token", "info"},
			},
			// Update and Read testing - full config
			{
				Config: `
resource "argocd_cluster" "this" {
	annotations = {
		test = "annotation"
	}

	labels = {
		test = "label"
	}

	name       = "foo"
	namespaces = ["default", "foo"]
	project    = "myproject1"
	server     = "https://kubernetes.default.svc.cluster.local"
	shard      = "1"

	config = {
		bearer_token = "abcdef.0123456789abcdef" # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.

		tls_client_config = {
			insecure = true
		}
	}
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_cluster.this", "config.tls_client_config.insecure", "true"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "annotations.%", "1"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "labels.%", "1"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "name", "foo"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "namespaces.0", "default"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "namespaces.1", "foo"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "project", "myproject1"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "server", "https://kubernetes.default.svc.cluster.local"),
					resource.TestCheckResourceAttr("argocd_cluster.this", "shard", "1"),
					// resource.TestCheckResourceAttr("argocd_cluster.this", "info.applications_count", "0"),
					// resource.TestCheckResourceAttr("argocd_cluster.this", "info.connection_state.status", "Successful"),
					// resource.TestCheckResourceAttr("argocd_cluster.this", "info.server_version", "1.24"),
				),
			},
			// ImportState testing - full config
			{
				ResourceName:            "argocd_cluster.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.bearer_token", "info"},
			},
		},
	})
}

func TestAccArgoCDCluster_TLSClientConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccArgoCDClusterTLSCertificate(t, acctest.RandString(10)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_cluster.tls", "info.0.connection_state.0.status", "Successful"),
					resource.TestCheckResourceAttr("argocd_cluster.tls", "info.0.server_version", "1.24"),
					resource.TestCheckResourceAttr("argocd_cluster.tls", "config.0.tls_client_config.0.insecure", "false"),
				),
			},
		},
	})
}

func TestAccArgoCDCluster_optionalName(t *testing.T) {
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccArgoCDClusterMetadataNoName(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "info.0.connection_state.0.status", "Successful"),
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "config.0.tls_client_config.0.insecure", "true"),
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "name", "https://kubernetes.default.svc.cluster.local"),
				),
			},
			{
				Config: testAccArgoCDClusterMetadata(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "info.0.connection_state.0.status", "Successful"),
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "config.0.tls_client_config.0.insecure", "true"),
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "name", name),
				),
			},
			{
				Config: testAccArgoCDClusterMetadataNoName(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "info.0.connection_state.0.status", "Successful"),
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "config.0.tls_client_config.0.insecure", "true"),
					resource.TestCheckResourceAttr("argocd_cluster.cluster_metadata", "name", "https://kubernetes.default.svc.cluster.local"),
				),
			},
		},
	})
}

func TestAccArgoCDCluster_invalidSameServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccArgoCDClusterTwiceWithSameServer(),
				ExpectError: regexp.MustCompile("cluster with server address .* already exists"),
			},
			{
				Config:      testAccArgoCDClusterTwiceWithSameServerNoNames(),
				ExpectError: regexp.MustCompile("cluster with server address .* already exists"),
			},
			{
				Config:      testAccArgoCDClusterTwiceWithSameLogicalServer(),
				ExpectError: regexp.MustCompile("cluster with server address .* already exists"),
			},
		},
	})
}

func TestAccArgoCDCluster_namespacesErrorWhenEmpty(t *testing.T) {
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccArgoCDClusterNamespacesContainsEmptyString(name),
				ExpectError: regexp.MustCompile("namespaces: must contain non-empty strings"),
			},
			{
				Config:      testAccArgoCDClusterNamespacesContainsEmptyString_MultipleItems(name),
				ExpectError: regexp.MustCompile("namespaces: must contain non-empty strings"),
			},
		},
	})
}

func testAccArgoCDClusterTLSCertificate(t *testing.T, clusterName string) string {
	rc, err := getInternalRestConfig()
	if err != nil {
		t.Error(err)
	}

	return fmt.Sprintf(`
resource "argocd_cluster" "tls" {
  server = "https://kubernetes.default.svc.cluster.local"
  name   = "%s"
  namespaces = ["bar", "baz"]
  config {
    tls_client_config {
      key_data    = <<EOT
%s
EOT
      cert_data   = <<EOT
%s
EOT
      ca_data     = <<EOT
%s
EOT
      server_name = "%s"
      insecure    = false
    }
  }
}
`, clusterName, rc.KeyData, rc.CertData, rc.CAData, rc.ServerName)
}

func testAccArgoCDClusterMetadata(clusterName string) string {
	return fmt.Sprintf(`
resource "argocd_cluster" "cluster_metadata" {
  server = "https://kubernetes.default.svc.cluster.local"
  name   = "%s"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
`, clusterName)
}

func testAccArgoCDClusterMetadataNoName() string {
	return `
resource "argocd_cluster" "cluster_metadata" {
  server = "https://kubernetes.default.svc.cluster.local"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
`
}

func testAccArgoCDClusterTwiceWithSameServer() string {
	return `
resource "argocd_cluster" "cluster_one_same_server" {
  server = "https://kubernetes.default.svc.cluster.local"
  name   = "foo"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
resource "argocd_cluster" "cluster_two_same_server" {
  server = "https://kubernetes.default.svc.cluster.local"
  name   = "bar"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}`
}

func testAccArgoCDClusterTwiceWithSameServerNoNames() string {
	return `
resource "argocd_cluster" "cluster_one_no_name" {
  server = "https://kubernetes.default.svc.cluster.local"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
resource "argocd_cluster" "cluster_two_no_name" {
  server = "https://kubernetes.default.svc.cluster.local"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
`
}

func testAccArgoCDClusterTwiceWithSameLogicalServer() string {
	return `
resource "argocd_cluster" "cluster_with_trailing_slash" {
  name = "server"
  server = "https://kubernetes.default.svc.cluster.local/"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
resource "argocd_cluster" "cluster_with_no_trailing_slash" {
  name = "server"
  server = "https://kubernetes.default.svc.cluster.local"
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}`
}

func testAccArgoCDClusterNamespacesContainsEmptyString(clusterName string) string {
	return fmt.Sprintf(`
resource "argocd_cluster" "simple" {
  server = "https://kubernetes.default.svc.cluster.local"
  name   = "%s"
  shard  = "1"
  namespaces = [""]
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
`, clusterName)
}

func testAccArgoCDClusterNamespacesContainsEmptyString_MultipleItems(clusterName string) string {
	return fmt.Sprintf(`
resource "argocd_cluster" "simple" {
  server = "https://kubernetes.default.svc.cluster.local"
  name   = "%s"
  shard  = "1"
  namespaces = ["default", ""]
  config {
    # Uses Kind's bootstrap token whose ttl is 24 hours after cluster bootstrap.
    bearer_token = "abcdef.0123456789abcdef"
    tls_client_config {
      insecure = true
    }
  }
}
`, clusterName)
}

// getInternalRestConfig returns the internal Kubernetes cluster REST config.
func getInternalRestConfig() (*rest.Config, error) {
	var kubeConfigFilePath string

	switch runtime.GOOS {
	case "windows":
		kubeConfigFilePath = fmt.Sprintf("%s\\.kube\\config", homedir.HomeDir())
	default:
		kubeConfigFilePath = fmt.Sprintf("%s/.kube/config", homedir.HomeDir())
	}

	cfg, err := clientcmd.LoadFromFile(kubeConfigFilePath)
	if err != nil {
		return nil, err
	}

	rc := &rest.Config{}

	for key, cluster := range cfg.Clusters {
		if key == "kind-argocd" {
			authInfo := cfg.AuthInfos[key]
			rc.Host = cluster.Server
			rc.ServerName = cluster.TLSServerName
			rc.TLSClientConfig.CAData = cluster.CertificateAuthorityData
			rc.TLSClientConfig.CertData = authInfo.ClientCertificateData
			rc.TLSClientConfig.KeyData = authInfo.ClientKeyData

			return rc, nil
		}
	}

	return nil, fmt.Errorf("could not find a kind-argocd cluster from the current ~/.kube/config file")
}
