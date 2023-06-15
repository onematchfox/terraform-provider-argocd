package planmodifiers

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ExpiredToken() planmodifier.Int64 {
	return &expiredToken{}
}

type expiredToken struct{}

func (*expiredToken) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var expiresAt types.Int64

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("expires_at"), &expiresAt)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Do not replace if expires_at is not set.
	if expiresAt.IsNull() || expiresAt.IsUnknown() {
		return
	}

	if expiresAt.ValueInt64() < time.Now().Unix() {
		// Token has expired - force recreation
		resp.RequiresReplace = true
		resp.PlanValue = types.Int64Unknown()
	}
}

func (d *expiredToken) Description(ctx context.Context) string {
	return "Checks to see if a token has expired and forces recreation of the resource if so."
}

func (d *expiredToken) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}
