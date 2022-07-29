package infoblox

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	ibclient "github.com/p2k3m/infoblox-go-client"
)

func resourceNetworkContainer() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"network_view": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of network view for the network container.",
			},
			"parent_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The parent network container block in cidr format to allocate from.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The network container's address, in CIDR format.",
			},
			"allocate_prefix_len": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Set the parameter's value > 0 to allocate next available network with corresponding prefix length from the network container defined by 'parent_cidr'",
			},
			"comment": {
				Type:        schema.TypeString,
				Default:     "",
				Optional:    true,
				Description: "A description of the network container.",
			},
			"ext_attrs": {
				Type:        schema.TypeString,
				Default:     "",
				Optional:    true,
				Description: "The Extensible attributes of the network container to be added/updated, as a map in JSON format",
			},
		},
	}
}

func resourceNetworkContainerCreate(d *schema.ResourceData, m interface{}, isIPv6 bool) error {
	nvName := d.Get("network_view").(string)
	parentCidr := d.Get("parent_cidr").(string)
	cidr := d.Get("cidr").(string)
	comment := d.Get("comment").(string)
	prefixLen := d.Get("allocate_prefix_len").(int)

	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}
	var tenantID string
	for attrName, attrValueInf := range extAttrs {
		attrValue, _ := attrValueInf.(string)
		if attrName == "Tenant ID" {
			tenantID = attrValue
			break
		}
	}
	if prefixLen < 2 {
		return fmt.Errorf(
			"prefixLen is less than 2")
	}

	// if cidr == "" || nvName == "" {
	// 	return fmt.Errorf(
	// 		"Tenant ID, network view's name and CIDR are required to create a network container")
	// }
	if nvName == "" {
		return fmt.Errorf(
			"Tenant ID, network view's name are required to create a network container")
	}

	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)
	if prefixLen < 7 {
		nc, err := objMgr.CreateNetworkContainer(nvName, cidr, isIPv6, comment, extAttrs)
		if err != nil {
			return fmt.Errorf(
				"creation of IPv6 network container block failed in network view '%s': %s",
				nvName, err.Error())
		}
		d.SetId(nc.Ref)

		return nil
	} else {
		_, err := objMgr.GetNetworkContainer(nvName, parentCidr, isIPv6, nil)
		if err != nil {
			return fmt.Errorf(
				"Allocation of network block within network container '%s' under network view '%s' failed: %s", parentCidr, nvName, err.Error())
		}

		nc, err := objMgr.AllocateNetworkContainer(nvName, parentCidr, isIPv6, uint(prefixLen), comment, extAttrs)
		if err != nil {
			return fmt.Errorf("Allocation of network block failed in network view (%s) : %s", nvName, err)
		}
		d.SetId(nc.Ref)
		return nil
	}
}

func resourceNetworkContainerRead(d *schema.ResourceData, m interface{}) error {
	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}
	var tenantID string
	tempVal, found := extAttrs["Tenant ID"]
	if found {
		tenantID = tempVal.(string)
	}

	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)

	obj, err := objMgr.GetNetworkContainerByRef(d.Id())
	if err != nil {
		return fmt.Errorf("failed to retrieve network container: %s", err.Error())
	}
	d.SetId(obj.Ref)

	return nil
}

func resourceNetworkContainerUpdate(d *schema.ResourceData, m interface{}) error {
	nvName := d.Get("network_view").(string)
	if d.HasChange("network_view") {
		return fmt.Errorf("changing the value of 'network_view' field is not allowed")
	}
	cidr := d.Get("cidr").(string)
	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}

	var tenantID string
	tempVal, found := extAttrs["Tenant ID"]
	if found {
		tenantID = tempVal.(string)
	}

	if cidr == "" || nvName == "" {
		return fmt.Errorf(
			"Tenant ID, network view's name and CIDR are required to update a network container")
	}

	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)

	comment := ""
	commentText, commentFieldFound := d.GetOk("comment")
	if commentFieldFound {
		comment = commentText.(string)
	}

	nc, err := objMgr.UpdateNetworkContainer(d.Id(), extAttrs, comment)
	if err != nil {
		return fmt.Errorf(
			"failed to update the network container in network view '%s': %s",
			nvName, err.Error())
	}

	d.SetId(nc.Ref)

	return nil
}

func resourceNetworkContainerDelete(d *schema.ResourceData, m interface{}) error {
	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}
	var tenantID string
	for attrName, attrValueInf := range extAttrs {
		attrValue, _ := attrValueInf.(string)
		if attrName == "Tenant ID" {
			tenantID = attrValue
			break
		}
	}
	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)

	if _, err := objMgr.DeleteNetworkContainer(d.Id()); err != nil {
		return fmt.Errorf(
			"deletion of the network container failed: %s", err.Error())
	}

	return nil
}

// TODO: implement this after infoblox-go-client refactoring
//func resourceNetworkContainerExists(d *schema.ResourceData, m interface{}, isIPv6 bool) (bool, error) {
//	return false, nil
//}

func resourceIPv4NetworkContainerCreate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerCreate(d, m, false)
}

func resourceIPv4NetworkContainerRead(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerRead(d, m)
}

func resourceIPv4NetworkContainerUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerUpdate(d, m)
}

func resourceIPv4NetworkContainerDelete(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerDelete(d, m)
}

//func resourceIPv4NetworkContainerExists(d *schema.ResourceData, m interface{}) (bool, error) {
//	return resourceNetworkContainerExists(d, m, false)
//}

func resourceIPv4NetworkContainer() *schema.Resource {
	nc := resourceNetworkContainer()
	nc.Create = resourceIPv4NetworkContainerCreate
	nc.Read = resourceIPv4NetworkContainerRead
	nc.Update = resourceIPv4NetworkContainerUpdate
	nc.Delete = resourceIPv4NetworkContainerDelete
	//nc.Exists = resourceIPv4NetworkContainerExists

	return nc
}

func resourceIPv6NetworkContainerCreate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerCreate(d, m, true)
}

func resourceIPv6NetworkContainerRead(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerRead(d, m)
}

func resourceIPv6NetworkContainerUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerUpdate(d, m)
}

func resourceIPv6NetworkContainerDelete(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerDelete(d, m)
}

//func resourceIPv6NetworkContainerExists(d *schema.ResourceData, m interface{}) (bool, error) {
//	return resourceNetworkContainerExists(d, m, true)
//}

func resourceIPv6NetworkContainer() *schema.Resource {
	nc := resourceNetworkContainer()
	nc.Create = resourceIPv6NetworkContainerCreate
	nc.Read = resourceIPv6NetworkContainerRead
	nc.Update = resourceIPv6NetworkContainerUpdate
	nc.Delete = resourceIPv6NetworkContainerDelete
	//nc.Exists = resourceIPv6NetworkContainerExists

	return nc
}
