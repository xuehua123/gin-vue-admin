package request

// PairingRegisterRequest defines the request body for registering a device role.
type PairingRegisterRequest struct {
	Role  string `json:"role" binding:"required,oneof=transmitter receiver"` // Role must be either 'transmitter' or 'receiver'
	Force bool   `json:"force"`                                              // Forcefully take over the role if it's already occupied
}
