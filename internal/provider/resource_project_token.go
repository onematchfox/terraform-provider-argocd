package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oboukili/terraform-provider-argocd/internal/diagnostics"
	"github.com/oboukili/terraform-provider-argocd/internal/sync"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &resourceProjectToken{}

func NewProjectTokenResource() resource.Resource {
	return &resourceProjectToken{}
}

type resourceProjectToken struct {
	si *ServerInterface
}

func (r *resourceProjectToken) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_token"
}

func (r *resourceProjectToken) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages ArgoCD project role JWT tokens. See [Project Roles](https://argo-cd.readthedocs.io/en/stable/user-guide/projects/#project-roles) for more info.",
		Attributes:          projectTokenSchemaAttributes(),
	}
}

func (r *resourceProjectToken) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceProjectToken) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data projectTokenModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.Project.ValueString()
	mutex := sync.GetProjectMutex(projectName)

	mutex.Lock()

	ctResp, err := r.si.ProjectClient.CreateToken(ctx, &project.ProjectTokenCreateRequest{
		Project:     projectName,
		Role:        data.Role.ValueString(),
		Description: data.Description.ValueString(),
		ExpiresIn:   int64(data.ExpiresIn.ValueDuration().Seconds()),
	})

	mutex.Unlock()

	if err != nil {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("create", "token for project", projectName, err)...)
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("created JWT token for project role %s", projectName))

	token, diags := newProjectToken(data, ctResp.GetToken())
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &token)...)
}

func (r *resourceProjectToken) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data projectTokenModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.Project.ValueString()
	mutex := sync.GetProjectMutex(projectName)

	// Delete token from state if project has been deleted in an out-of-band fashion
	mutex.RLock()
	defer mutex.RUnlock()

	p, err := r.si.ProjectClient.Get(ctx, &project.ProjectQuery{
		Name: projectName,
	})

	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("read", "project", projectName, err)...)
			return
		}

		// Project has been deleted in an out-of-band fashion
		data = projectTokenModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	_, _, err = p.GetJWTToken(
		data.Role.ValueString(),
		0,
		data.ID.ValueString(),
	)

	if err != nil {
		// Token has been deleted in an out-of-band fashion
		data = projectTokenModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceProjectToken) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data projectTokenModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// noop

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceProjectToken) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data projectTokenModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.Project.ValueString()
	mutex := sync.GetProjectMutex(projectName)

	mutex.Lock()

	_, err := r.si.ProjectClient.DeleteToken(ctx, &project.ProjectTokenDeleteRequest{
		Project: projectName,
		Role:    data.Role.ValueString(),
		Id:      data.ID.ValueString(),
	})

	mutex.Unlock()

	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("delete", "token for project", projectName, err)...)
		return
	}
}
