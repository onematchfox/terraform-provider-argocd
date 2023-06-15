package planmodifiers

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	customtypes "github.com/oboukili/terraform-provider-argocd/internal/types"
)

func AutoRenewingToken() planmodifier.Int64 {
	return &autoRenewingToken{}
}

type autoRenewingToken struct{}

func (*autoRenewingToken) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Fetch renew_after
	var renewAfter customtypes.Duration

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("renew_after"), &renewAfter)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Do not replace if renew_after is not set.
	if renewAfter.IsNull() || renewAfter.IsUnknown() {
		return
	}

	// Fetch issued_at
	var issuedAt types.Int64

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("issued_at"), &issuedAt)...) // note: fetch from state not plan

	if resp.Diagnostics.HasError() {
		return
	}

	// Do not replace if issued_at is not set as we can't determine how old the token is.
	// This should never happen as all tokens have an issue date - so just being on the safe side.
	if issuedAt.IsNull() || issuedAt.IsUnknown() {
		resp.Diagnostics.AddWarning(
			"Token has no issue date set",
			"`issued_at` was not set on this token. This should never happen. Please report this to the provider developers.")

		return
	}

	tokenAge := time.Now().Unix() - issuedAt.ValueInt64()

	if tokenAge > int64(renewAfter.ValueDuration().Seconds()) {
		// Token is older than renewAfter - force recreation
		resp.RequiresReplace = true
		resp.PlanValue = types.Int64Unknown()
	}
}

func (d *autoRenewingToken) Description(ctx context.Context) string {
	return "Checks to see if a token has expired and forces recreation of the resource if so."
}

func (d *autoRenewingToken) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}
