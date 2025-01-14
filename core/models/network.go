package models

// NetworkInterface represents a network interface for the microvm.
type NetworkInterface struct {
	// GuestDeviceName is the name of the network interface to create in the microvm.
	GuestDeviceName string `json:"guest_device_name" validate:"required,excludesall=/@,guestDeviceName"`
	// AllowMetadataRequests indicates that this interface can be used for metadata requests.
	// TODO: we may hide this within the firecracker plugin.
	AllowMetadataRequests bool `json:"allow_mmds,omitempty"`
	// GuestMAC allows the specifying of a specifi MAC address to use for the interface. If
	// not supplied a autogenerated MAC address will be used.
	GuestMAC string `json:"guest_mac,omitempty" validate:"omitempty,mac"`
	// Type is the type of host network interface type to create to use by the guest.
	Type IfaceType `json:"type" validate:"oneof=tap macvtap unsupported"`
	// Address is an optional IP address to assign to this interface. If not supplied then DHCP will be used.
	Address string `json:"address,omitempty" validate:"omitempty,cidr"`
	// TODO: add rate limiting.
	// TODO: add CNI.
}

type NetworkInterfaceStatus struct {
	// HostDeviceName is the name of the network interface used from the host. This will be
	// a tuntap or macvtap interface.
	HostDeviceName string `json:"host_device_name"`
	// Index is the index of the network interface on the host.
	Index int `json:"index"`
	// MACAddress is the MAC address of the host interface.
	MACAddress string `json:"mac_address"`
}

// NetworkInterfaceStatuses is a collection of network interfaces.
type NetworkInterfaceStatuses map[string]*NetworkInterfaceStatus

// IfaceType is a type representing the supported network interface types.
type IfaceType string

const (
	// IfaceTypeTap is a TAP network interface.
	IfaceTypeTap = "tap"
	// IfaceTypeMacvtap is a MACVTAP network interface.
	IfaceTypeMacvtap = "macvtap"
	// IfaceTypeUnsupported is a type that represents an unsupported network interface type.
	IfaceTypeUnsupported = "unsupported"
)
