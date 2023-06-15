package provider

import (
	"encoding/json"

	"github.com/cristalhq/jwt/v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/oboukili/terraform-provider-argocd/internal/diagnostics"
	customplanmodifiers "github.com/oboukili/terraform-provider-argocd/internal/planmodifiers"
	customtypes "github.com/oboukili/terraform-provider-argocd/internal/types"
	"github.com/oboukili/terraform-provider-argocd/internal/utils"
)

type accountTokenModel struct {
	Account    types.String         `tfsdk:"account"`
	ExpiresAt  types.Int64          `tfsdk:"expires_at"`
	ExpiresIn  customtypes.Duration `tfsdk:"expires_in"`
	ID         types.String         `tfsdk:"id"`
	IssuedAt   types.Int64          `tfsdk:"issued_at"`
	JWT        types.String         `tfsdk:"jwt"`
	RenewAfter customtypes.Duration `tfsdk:"renew_after"`
}

func accountTokenSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"account": schema.StringAttribute{
			MarkdownDescription: "Account name. Defaults to the current account. I.e. the account configured on the `provider` block.",
			Optional:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"expires_in": schema.StringAttribute{
			MarkdownDescription: "Duration before the token will expire. Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. E.g. `12h`, `7d`. Default: No expiration.",
			Optional:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			CustomType: customtypes.DurationType,
		},
		"renew_after": schema.StringAttribute{
			MarkdownDescription: "Duration to control token silent regeneration based on token age. Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. If set, then the token will be regenerated if it is older than `renew_after`. I.e. if `currentDate - issued_at > renew_after`.",
			Optional:            true,
			CustomType:          customtypes.DurationType,
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "Token identifier",
			Computed:            true,
		},
		"jwt": schema.StringAttribute{
			MarkdownDescription: "The raw JWT.",
			Computed:            true,
			Sensitive:           true,
		},
		"issued_at": schema.Int64Attribute{
			MarkdownDescription: "Unix timestamp at which the token was issued.",
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				customplanmodifiers.AutoRenewingToken(),
			},
		},
		"expires_at": schema.Int64Attribute{
			MarkdownDescription: "If `expires_in` is set, Unix timestamp upon which the token will expire.",
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				customplanmodifiers.ExpiredToken(),
			},
		},
	}
}

func newAccountToken(m accountTokenModel, t string) (*accountTokenModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	token, err := jwt.ParseString(t)
	if err != nil {
		diags.Append(diagnostics.Error("Account token is not a valid jwt", err)...)
		return nil, diags
	}

	var claims jwt.StandardClaims

	if err = json.Unmarshal(token.RawClaims(), &claims); err != nil {
		diags.Append(diagnostics.Error("Token claims for account token could not be parsed", err)...)
		return nil, diags
	}

	return &accountTokenModel{
		Account:    m.Account,
		ExpiresAt:  utils.OptionalNumericDateValue(claims.ExpiresAt),
		ExpiresIn:  m.ExpiresIn,
		ID:         types.StringValue(claims.ID),
		IssuedAt:   utils.OptionalNumericDateValue(claims.IssuedAt),
		JWT:        types.StringValue(token.String()),
		RenewAfter: m.RenewAfter,
	}, diags
}
