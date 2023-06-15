package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/account"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/session"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oboukili/terraform-provider-argocd/internal/diagnostics"
	"github.com/oboukili/terraform-provider-argocd/internal/sync"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &accountTokenResource{}

func NewAccountTokenResource() resource.Resource {
	return &accountTokenResource{}
}

type accountTokenResource struct {
	si *ServerInterface
}

func (r *accountTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_token"
}

func (r *accountTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages ArgoCD [account](https://argo-cd.readthedocs.io/en/latest/user-guide/commands/_account/) JWT tokens.",
		Attributes:          accountTokenSchemaAttributes(),
	}
}

func (r *accountTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accountTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data accountTokenModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	accountName, d := r.getAccount(ctx, data)
	if d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	sync.SecretsMutex.Lock()

	ctResp, err := r.si.AccountClient.CreateToken(ctx, &account.CreateTokenRequest{
		Name:      accountName,
		ExpiresIn: int64(data.ExpiresIn.ValueDuration().Seconds()),
	})

	sync.SecretsMutex.Unlock()

	if err != nil {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("create", "token for account", accountName, err)...)
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("created JWT token for account %s", accountName))

	token, diags := newAccountToken(data, ctResp.GetToken())
	resp.Diagnostics.Append(diags...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &token)...)
}

func (r *accountTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data accountTokenModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	accountName, d := r.getAccount(ctx, data)
	if d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	sync.ConfigurationMutex.RLock() // Yes, this is a different mutex than used elsewhere in this resource - accounts are stored in `argocd-cm` whereas tokens are stored in `argocd-secret`
	defer sync.ConfigurationMutex.RUnlock()

	_, err := r.si.AccountClient.GetAccount(ctx, &account.GetAccountRequest{
		Name: accountName,
	})

	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("read", "account", accountName, err)...)
			return
		}

		// Clear state if account has been deleted in an out-of-band fashion
		data = accountTokenModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *accountTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data accountTokenModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// noop

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *accountTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data accountTokenModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Initialize API clients
	resp.Diagnostics.Append(r.si.InitClients(ctx)...)

	// Check for errors before proceeding
	if resp.Diagnostics.HasError() {
		return
	}

	accountName, d := r.getAccount(ctx, data)
	if d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	sync.SecretsMutex.Lock()

	_, err := r.si.AccountClient.DeleteToken(ctx, &account.DeleteTokenRequest{
		Name: accountName,
		Id:   data.ID.ValueString(),
	})

	sync.SecretsMutex.Unlock()

	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		resp.Diagnostics.Append(diagnostics.ArgoCDAPIError("delete", "token for account", accountName, err)...)
		return
	}
}

func (r *accountTokenResource) getAccount(ctx context.Context, m accountTokenModel) (string, diag.Diagnostics) {
	accountName := m.Account.ValueString()
	if len(accountName) > 0 {
		return accountName, nil
	}

	userInfo, err := r.si.SessionClient.GetUserInfo(ctx, &session.GetUserInfoRequest{})
	if err != nil {
		return "", diagnostics.Error("Failed to get current account", err)
	} else if userInfo == nil || userInfo.Username == "" {
		return "", []diag.Diagnostic{
			diag.NewErrorDiagnostic(
				"Failed to get current account",
				"The  server did not return a response on the user info endpoint. This usually indicates that the provider is configured with `core=true`. In this case, the `account` attribute on `_account_token` resources must configured.",
			),
		}
	}

	return userInfo.Username, nil
}
