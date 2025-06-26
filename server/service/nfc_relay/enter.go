package nfc_relay

type ServiceGroup struct {
	TransactionService NFCTransactionService
	MqttService        MqttService
	PairingPoolService PairingPoolService
}
