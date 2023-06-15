package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oboukili/terraform-provider-argocd/internal/diagnostics"
	"github.com/oboukili/terraform-provider-argocd/internal/sync"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &clusterResource{}

func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

// clusterResource defines the resource implementation.
type clusterResource struct {
	si *ServerInterface
}

func (r *clusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages [clusters](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#clusters) within ArgoCD.",
		Attributes:          schemaClusterAttributes(),
	}
}

func (r *clusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	si, ok := req.ProviderData.(*ServerInterface)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *ServerInterface, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.si = si
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data clusterModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	// Need a full lock here to avoid race conditions between List existing clusters and creating a new one
	sync.ClustersMutex.Lock()
	defer sync.ClustersMutex.Unlock()

	// Cluster are unique by "server address" so we should check there is no existing cluster with this address before
	existingClusters, err := r.si.ClusterClient.List(ctx, &cluster.ClusterQuery{
		Id: &cluster.ClusterID{
			Type:  "server",
			Value: data.Server.ValueString(), // TODO: not used by backend, upstream bug ?
		},
	})

	if err != nil {
		resp.Diagnostics.Append(diagnostics.Error(fmt.Sprintf("failed to list existing clusters when creating cluster %s", data.Server.ValueString()), err)...)
	}

	rtrimmedServer := strings.TrimRight(data.Server.ValueString(), "/")

	if len(existingClusters.Items) > 0 {
		for _, existingCluster := range existingClusters.Items {
			if rtrimmedServer == strings.TrimRight(existingCluster.Server, "/") {
				resp.Diagnostics.AddError(fmt.Sprintf("cluster with server address %s already exists", data.Server.ValueString()), "")
				return
			}
		}
	}

	c, err := r.si.ClusterClient.Create(ctx, &cluster.ClusterCreateRequest{
		Cluster: data.Cluster(),
		Upsert:  false})

	if err != nil {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("create", "cluster", data.Server.ValueString(), err)...)
		return
	}

	if c == nil {
		resp.Diagnostics.AddError("unexpected response when creating ArgoCD Cluster - no cluster created", "")
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("created cluster %s", data.Server.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, data.FromCluster(c))...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data clusterModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	sync.ClustersMutex.RLock()
	defer sync.ClustersMutex.RUnlock()

	c, err := r.si.ClusterClient.Get(ctx, data.ClusterQuery())

	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("read", "cluster", data.Server.ValueString(), err)...)
			return
		}

		// Clear state if account has been deleted in an out-of-band fashion
		data = clusterModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data.FromCluster(c))...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data clusterModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	sync.ClustersMutex.Lock()

	c, err := r.si.ClusterClient.Update(ctx, &cluster.ClusterUpdateRequest{
		Cluster: data.Cluster(),
	})

	sync.ClustersMutex.Unlock()

	if err != nil {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("update", "cluster", data.Server.ValueString(), err)...)
	}

	if c == nil {
		resp.Diagnostics.AddError("unexpected response when creating ArgoCD Cluster - no cluster created", "")
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("updated cluster %s", data.Server.ValueString()))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data.FromCluster(c))...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data clusterModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	sync.ClustersMutex.Lock()

	_, err := r.si.ClusterClient.Delete(ctx, data.ClusterQuery())

	sync.ClustersMutex.Unlock()

	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("delete", "cluster", data.Server.ValueString(), err)...)
		return
	}
}

func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
