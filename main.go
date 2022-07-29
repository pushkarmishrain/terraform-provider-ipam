package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/p2k3m/terraform-provider-infoblox/infoblox"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: infoblox.Provider})
}
