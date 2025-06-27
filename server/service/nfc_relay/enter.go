package nfc_relay

import "sync"

type ServiceGroup struct {
	transactionService *NFCTransactionService
	mqttService        *MqttService
	pairingPoolService *PairingPoolService
}

var (
	oncePairing     sync.Once
	onceMqtt        sync.Once
	onceTransaction sync.Once
)

var ServiceGroupApp = new(ServiceGroup)

func (s *ServiceGroup) PairingPoolService() *PairingPoolService {
	oncePairing.Do(func() {
		s.pairingPoolService = NewPairingPoolService()
	})
	return s.pairingPoolService
}

func (s *ServiceGroup) MqttService() *MqttService {
	onceMqtt.Do(func() {
		s.mqttService = &MqttService{}
	})
	return s.mqttService
}

func (s *ServiceGroup) TransactionService() *NFCTransactionService {
	onceTransaction.Do(func() {
		s.transactionService = new(NFCTransactionService)
	})
	return s.transactionService
}
