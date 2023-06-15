package provider

import (
	"fmt"
	"strings"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/elliotchance/pie/v2"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/oboukili/terraform-provider-argocd/internal/utils"
	"github.com/oboukili/terraform-provider-argocd/internal/validators"
)

type clusterModel struct {
	ID          types.String            `tfsdk:"id"`
	Annotations map[string]types.String `tfsdk:"annotations"`
	Config      *clusterConfig          `tfsdk:"config"`
	Info        *clusterInfo            `tfsdk:"info"`
	Labels      map[string]types.String `tfsdk:"labels"`
	Name        types.String            `tfsdk:"name"`
	Namespaces  []types.String          `tfsdk:"namespaces"`
	Project     types.String            `tfsdk:"project"`
	Server      types.String            `tfsdk:"server"`
	Shard       types.Int64             `tfsdk:"shard"`
}

func (m clusterModel) Cluster() *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		Annotations: utils.MapMap(m.Annotations, utils.ValueString),
		Config:      m.Config.ClusterConfig(),
		Labels:      utils.MapMap(m.Labels, utils.ValueString),
		Name:        m.Name.ValueString(),
		Namespaces:  pie.Map(m.Namespaces, utils.ValueString),
		Project:     m.Project.ValueString(),
		Server:      m.Server.ValueString(),
		Shard:       utils.OptionalInt64(m.Shard),
	}
}

func (m clusterModel) ClusterQuery() *cluster.ClusterQuery {
	cq := &cluster.ClusterQuery{}

	id := strings.Split(strings.TrimPrefix(m.ID.ValueString(), "https://"), "/")
	if len(id) > 1 {
		cq.Name = id[len(id)-1]
		cq.Server = fmt.Sprintf("https://%s", strings.Join(id[:len(id)-1], "/"))
	} else {
		cq.Server = m.ID.ValueString()
	}

	return cq
}

func (m clusterModel) GetID() string {
	if !m.ID.IsNull() && !m.ID.IsUnknown() {
		return m.ID.ValueString()
	}

	if m.Name.ValueString() != "" && !m.Name.Equal(m.Server) {
		return fmt.Sprintf("%s/%s", m.Server.ValueString(), m.Name.ValueString())
	}

	return m.Server.ValueString()
}

func (m clusterModel) FromCluster(c *v1alpha1.Cluster) *clusterModel {
	if c == nil {
		return nil
	}

	cm := &clusterModel{
		Annotations: utils.MapMap(c.Annotations, types.StringValue),
		Config:      newClusterConfig(c.Config, m.Config),
		ID:          types.StringValue(m.GetID()),
		Info:        newClusterInfo(c.Info),
		Labels:      utils.MapMap(c.Labels, types.StringValue),
		Namespaces:  pie.Map(c.Namespaces, types.StringValue),
		Project:     utils.OptionalStringValue(c.Project),
		Server:      utils.OptionalStringValue(c.Server),
		Shard:       utils.OptionalInt64Value(c.Shard),
	}

	if !m.Name.IsNull() || c.Server != c.Name {
		// If name was provided in config or is non default (not equal to Server) then we should track it
		cm.Name = utils.OptionalStringValue(c.Name)
	}

	return cm
}

func schemaClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"annotations": schema.MapAttribute{
			MarkdownDescription: "An unstructured key value map stored with the cluster secret that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations",
			Optional:            true,
			ElementType:         types.StringType,
			Validators: []validator.Map{
				validators.MetadataAnnotations(),
			},
		},
		"config": clusterConfigSchemaAttribute(),
		"id": schema.StringAttribute{
			MarkdownDescription: "Token identifier",
			Computed:            true,
		},
		"info": clusterInfoSchemaAttribute(),
		"labels": schema.MapAttribute{
			MarkdownDescription: "Map of string keys and values that can be used to organize and categorize (scope and select) the cluster secret. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels",
			Optional:            true,
			ElementType:         types.StringType,
			Validators: []validator.Map{
				validators.MetadataLabels(),
			},
		},
		"name": schema.StringAttribute{
			Description: "Name of the cluster. If omitted, will use the server address.",
			Optional:    true,
			// DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
			// 	if k == "name" {
			// 		name, nameOk := d.GetOk("name")
			// 		server, serverOk := d.GetOk("server")
			// 		// Actual value is same as 'server' but not explicitly set
			// 		if nameOk && serverOk && name == server && oldValue == server && newValue == "" {
			// 			return true
			// 		}
			// 	}
			// 	return false
			// },
		},
		"namespaces": schema.SetAttribute{
			Description: "List of namespaces which are accessible in that cluster. Cluster level resources would be ignored if namespace list is not empty.",
			Optional:    true,
			ElementType: types.StringType,
		},
		"project": schema.StringAttribute{
			Description: "Reference between project and cluster that allow you automatically to be added as item inside Destinations project entity. More info: https://argo-cd.readthedocs.io/en/stable/user-guide/projects/#project-scoped-repositories-and-clusters.",
			Optional:    true,
		},
		"server": schema.StringAttribute{
			Description: "Server is the API server URL of the Kubernetes cluster.",
			Optional:    true,
		},
		"shard": schema.Int64Attribute{
			Description: "Optional shard number. Calculated on the fly by the application controller if not specified.",
			Optional:    true,
		},
	}
}

type clusterConfig struct {
	AWSAuthConfig      *awsAuthConfig      `tfsdk:"aws_auth_config"`
	BearerToken        types.String        `tfsdk:"bearer_token"`
	ExecProviderConfig *execProviderConfig `tfsdk:"exec_provider_config"`
	Password           types.String        `tfsdk:"password"`
	TLSClientConfig    *tlsClientConfig    `tfsdk:"tls_client_config"`
	Username           types.String        `tfsdk:"username"`
}

func (c *clusterConfig) ClusterConfig() v1alpha1.ClusterConfig {
	return v1alpha1.ClusterConfig{
		AWSAuthConfig:      c.AWSAuthConfig.AWSAuthConfig(),
		BearerToken:        c.BearerToken.ValueString(),
		ExecProviderConfig: c.ExecProviderConfig.ExecProviderConfig(),
		Password:           c.Password.ValueString(),
		Username:           c.Username.ValueString(),
		TLSClientConfig:    c.TLSClientConfig.TLSClientConfig(),
	}
}

func newClusterConfig(cc v1alpha1.ClusterConfig, state *clusterConfig) *clusterConfig {
	config := &clusterConfig{
		AWSAuthConfig: newAWSAuthConfig(cc.AWSAuthConfig),
		Username:      utils.OptionalStringValue(cc.Username),
	}

	if state != nil {
		// Sensitive values not returned by API - load from existing state if possible
		config.BearerToken = state.BearerToken
		config.Password = state.Password

		config.ExecProviderConfig = newExecProviderConfig(cc.ExecProviderConfig, state.ExecProviderConfig)
		config.TLSClientConfig = newTLSClientConfig(cc.TLSClientConfig, state.TLSClientConfig)
	} else {
		config.ExecProviderConfig = newExecProviderConfig(cc.ExecProviderConfig, nil)
		config.TLSClientConfig = newTLSClientConfig(cc.TLSClientConfig, nil)
	}

	return config
}

func clusterConfigSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Description: "Cluster information for connecting to a cluster.",
		Required:    true,
		Attributes: map[string]schema.Attribute{
			"aws_auth_config": awsAuthConfigSchemaAttribute(),
			"bearer_token": schema.StringAttribute{
				Description: "Server requires Bearer authentication. The client will not attempt to use refresh tokens for an OAuth2 flow.",
				Optional:    true,
				Sensitive:   true,
			},
			"exec_provider_config": execProviderConfigSchemaAttribute(),
			"password": schema.StringAttribute{
				Description: "Password for servers that require Basic authentication.",
				Optional:    true,
				Sensitive:   true,
			},
			"tls_client_config": tlsClientConfigSchemaAttribute(),
			"username": schema.StringAttribute{
				Description: "Username for servers that require Basic authentication.",
				Optional:    true,
			},
		},
	}
}

type awsAuthConfig struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	RoleARN     types.String `tfsdk:"role_arn"`
}

func (c *awsAuthConfig) AWSAuthConfig() *v1alpha1.AWSAuthConfig {
	if c == nil {
		return nil
	}

	return &v1alpha1.AWSAuthConfig{
		ClusterName: c.ClusterName.ValueString(),
		RoleARN:     c.RoleARN.ValueString(),
	}
}

func newAWSAuthConfig(aac *v1alpha1.AWSAuthConfig) *awsAuthConfig {
	if aac == nil {
		return nil
	}

	return &awsAuthConfig{
		ClusterName: utils.OptionalStringValue(aac.ClusterName),
		RoleARN:     utils.OptionalStringValue(aac.RoleARN),
	}
}

func awsAuthConfigSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Description: "AWS cluster name.",
				Optional:    true,
			},
			"role_arn": schema.StringAttribute{
				Description: "IAM role ARN. If set then AWS IAM Authenticator assume a role to perform cluster operations instead of the default AWS credential provider chain.",
				Optional:    true,
			},
		},
	}
}

type execProviderConfig struct {
	APIVersion  types.String            `tfsdk:"api_version"`
	Args        []types.String          `tfsdk:"args"`
	Command     types.String            `tfsdk:"command"`
	Env         map[string]types.String `tfsdk:"env"`
	InstallHint types.String            `tfsdk:"install_hint"`
}

func (c *execProviderConfig) ExecProviderConfig() *v1alpha1.ExecProviderConfig {
	if c == nil {
		return nil
	}

	return &v1alpha1.ExecProviderConfig{
		APIVersion:  c.APIVersion.ValueString(),
		Args:        pie.Map(c.Args, utils.ValueString),
		Command:     c.Command.ValueString(),
		Env:         utils.MapMap(c.Env, utils.ValueString),
		InstallHint: c.InstallHint.ValueString(),
	}
}

func newExecProviderConfig(epc *v1alpha1.ExecProviderConfig, state *execProviderConfig) *execProviderConfig {
	if epc == nil {
		return nil
	}

	c := &execProviderConfig{
		APIVersion:  utils.OptionalStringValue(epc.APIVersion),
		Command:     utils.OptionalStringValue(epc.Command),
		InstallHint: utils.OptionalStringValue(epc.InstallHint),
	}

	if state != nil {
		// Sensitive values not returned by API - load from existing state if available
		c.Args = state.Args
		c.Env = state.Env
	}

	return c
}

func execProviderConfigSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Description: "Configuration for an exec provider used to call an external command to perform cluster authentication See: https://godoc.org/k8s.io/client-go/tools/clientcmd/api#ExecConfig.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				Description: "Preferred input version of the ExecInfo",
				Optional:    true,
			},
			"args": schema.ListAttribute{
				Description: "Arguments to pass to the command when executing it",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"command": schema.StringAttribute{
				Description: "Command to execute",
				Optional:    true,
			},
			"env": schema.MapAttribute{
				Description: "Env defines additional environment variables to expose to the process. Passed as a map of strings",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"install_hint": schema.StringAttribute{
				Description: "This text is shown to the user when the executable doesn't seem to be present",
				Optional:    true,
			},
		},
	}
}

type tlsClientConfig struct {
	CAData     types.String `tfsdk:"ca_data"`
	CertData   types.String `tfsdk:"cert_data"`
	Insecure   types.Bool   `tfsdk:"insecure"`
	KeyData    types.String `tfsdk:"key_data"`
	ServerName types.String `tfsdk:"server_name"`
}

func (c *tlsClientConfig) TLSClientConfig() v1alpha1.TLSClientConfig {
	if c == nil {
		return v1alpha1.TLSClientConfig{}
	}

	return v1alpha1.TLSClientConfig{
		CAData:     []byte(c.CAData.ValueString()),
		CertData:   []byte(c.CertData.ValueString()),
		Insecure:   c.Insecure.ValueBool(),
		KeyData:    []byte(c.KeyData.ValueString()),
		ServerName: c.ServerName.ValueString(),
	}
}

func newTLSClientConfig(tcc v1alpha1.TLSClientConfig, state *tlsClientConfig) *tlsClientConfig {
	c := &tlsClientConfig{
		CAData:     utils.OptionalByteSliceValue(tcc.CAData),
		CertData:   utils.OptionalByteSliceValue(tcc.CertData),
		Insecure:   types.BoolValue(tcc.Insecure),
		ServerName: utils.OptionalStringValue(tcc.ServerName),
	}

	if state != nil {
		// Sensitive value not returned by API - load from existing state if available
		c.KeyData = state.KeyData
	}

	return c
}

func tlsClientConfigSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Description: "Settings to enable transport layer security when connecting to the cluster.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"ca_data": schema.StringAttribute{
				Description: "PEM-encoded bytes (typically read from a root certificates bundle).",
				Optional:    true,
			},
			"cert_data": schema.StringAttribute{
				Description: "PEM-encoded bytes (typically read from a client certificate file).",
				Optional:    true,
			},
			"insecure": schema.BoolAttribute{
				Description: "Whether server should be accessed without verifying the TLS certificate.",
				Optional:    true,
			},
			"key_data": schema.StringAttribute{
				Description: "PEM-encoded bytes (typically read from a client certificate key file).",
				Optional:    true,
				Sensitive:   true,
			},
			"server_name": schema.StringAttribute{
				Description: "Name to pass to the server for SNI and used in the client to check server certificates against. If empty, the hostname used to contact the server is used.",
				Optional:    true,
			},
		},
	}
}

type clusterInfo struct {
	ApplicationsCount types.Int64      `tfsdk:"applications_count"`
	ConnectionState   *connectionState `tfsdk:"connection_state"`
	ServerVersion     types.String     `tfsdk:"server_version"`
}

func newClusterInfo(ci v1alpha1.ClusterInfo) *clusterInfo {
	return &clusterInfo{
		ApplicationsCount: types.Int64Value(ci.ApplicationsCount),
		ConnectionState:   newConnectionState(ci.ConnectionState),
		ServerVersion:     types.StringValue(ci.ServerVersion),
	}
}

func clusterInfoSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Description: "Information about cluster cache and state.",
		Computed:    true,
		Attributes: map[string]schema.Attribute{
			"applications_count": schema.Int64Attribute{
				Description: "Number of applications managed by Argo CD on the cluster.",
				Computed:    true,
			},
			"connection_state": connectionStateSchemaAttribute(),
			"server_version": schema.StringAttribute{
				Description: "Kubernetes version of the cluster.",
				Computed:    true,
			},
		},
	}
}

type connectionState struct {
	Message types.String `tfsdk:"message"`
	Status  types.String `tfsdk:"status"`
}

func newConnectionState(cs v1alpha1.ConnectionState) *connectionState {
	return &connectionState{
		Message: types.StringValue(cs.Message),
		Status:  types.StringValue(cs.Status),
	}
}

func connectionStateSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		Description: "Information about the connection to the cluster.",
		Computed:    true,
		Attributes: map[string]schema.Attribute{
			"message": schema.StringAttribute{
				Description: "Human readable information about the connection status.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status indicator for the connection.",
				Computed:    true,
			},
		},
	}
}
