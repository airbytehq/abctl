package auth

// Cmd represents the auth command group
type Cmd struct {
	Login              LoginCmd              `cmd:"" help:"Login to Airbyte using OAuth."`
	Logout             LogoutCmd             `cmd:"" help:"Logout and clear stored credentials."`
	SwitchOrganization SwitchOrganizationCmd `cmd:"" aliases:"switch" help:"Switch to a different organization."`
}
